import { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import {
  Button, Input, Space, Typography, Drawer, Descriptions, Divider,
  Tooltip, Dropdown, Segmented, notification, Tag,
  Modal, Form, Select,
} from 'antd';
import {
  ApartmentOutlined, FolderOutlined, CopyOutlined,
  PlusOutlined, ReloadOutlined, SearchOutlined, MoreOutlined,
  DeleteOutlined, EditOutlined, FolderOpenOutlined,
  ExclamationCircleOutlined, UserOutlined, TeamOutlined,
  DesktopOutlined, ContactsOutlined, ContainerOutlined,
  RightOutlined, DownOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';

const { Text, Title } = Typography;

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface OU {
  dn: string;
  name: string;
  description: string;
  childCount: number;
}

interface OUChild {
  dn: string;
  name: string;
  objectClass: string;
  description?: string;
}

interface TreeNode {
  dn: string;
  name: string;
  objectClass: string;
  description?: string;
}

type ViewMode = 'list' | 'tree';

const childTypeIcon = (cls: string) => {
  switch (cls) {
    case 'user': return <UserOutlined />;
    case 'group': return <TeamOutlined />;
    case 'computer': return <DesktopOutlined />;
    case 'contact': return <ContactsOutlined />;
    case 'ou': return <FolderOutlined />;
    case 'container': return <ContainerOutlined />;
    default: return <FolderOutlined />;
  }
};

const childTypeColor = (cls: string) => {
  switch (cls) {
    case 'user': return 'blue';
    case 'group': return 'green';
    case 'computer': return 'purple';
    case 'contact': return 'orange';
    case 'ou': return 'cyan';
    default: return 'default';
  }
};

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

const cnFromDN = (dn: string) =>
  dn.split(',')[0]?.replace(/^(CN|OU)=/i, '') || dn;

const parentDN = (dn: string) =>
  dn.split(',').slice(1).join(',');

const MONO = { fontFamily: "'JetBrains Mono', monospace", fontSize: 12 };

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

/** Navigate path for a tree leaf object */
function objectNavPath(cls: string): string | null {
  switch (cls) {
    case 'user': return '/users';
    case 'group': return '/groups';
    case 'computer': return '/computers';
    case 'contact': return '/contacts';
    default: return null;
  }
}

/* ------------------------------------------------------------------ */
/*  Custom Directory Tree                                              */
/* ------------------------------------------------------------------ */

interface DirNode {
  dn: string;
  name: string;
  type: 'ou' | 'container' | 'user' | 'group' | 'computer' | 'contact' | 'unknown';
  childCount?: number;
  children?: DirNode[];
}

function buildDirTree(
  treeMap: Record<string, OU[]>,
  contents: Record<string, TreeNode[]>,
  parentKey: string,
): DirNode[] {
  const subOUs = treeMap[parentKey] || [];

  const ouNodes: DirNode[] = subOUs.map((ou) => ({
    dn: ou.dn,
    name: ou.name,
    type: 'ou' as const,
    childCount: ou.childCount,
    children: [
      ...buildDirTree(treeMap, contents, ou.dn),
      ...(contents[ou.dn] || []).map((obj) => ({
        dn: obj.dn,
        name: obj.name,
        type: (obj.objectClass || 'unknown') as DirNode['type'],
      })),
    ],
  }));

  // If no sub-OUs for this key, return leaf objects directly
  if (subOUs.length === 0) {
    return (contents[parentKey] || []).map((obj) => ({
      dn: obj.dn,
      name: obj.name,
      type: (obj.objectClass || 'unknown') as DirNode['type'],
    }));
  }

  return ouNodes;
}

function DirectoryTree({
  nodes,
  depth = 0,
  onSelectOU,
  onSelectObject,
}: {
  nodes: DirNode[];
  depth?: number;
  onSelectOU: (dn: string) => void;
  onSelectObject: (dn: string, type: string) => void;
}) {
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const toggle = (dn: string) => {
    setCollapsed((prev) => ({ ...prev, [dn]: !prev[dn] }));
  };

  return (
    <>
      {nodes.map((node) => {
        const hasChildren = node.children && node.children.length > 0;
        const isCollapsed = collapsed[node.dn] ?? false;
        const isContainer = node.type === 'ou' || node.type === 'container';

        return (
          <div key={node.dn}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                paddingLeft: depth * 20,
                padding: '3px 4px 3px ' + (depth * 20 + 4) + 'px',
                borderRadius: 4,
                cursor: 'pointer',
                transition: 'background 0.15s',
              }}
              onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--ant-color-fill-content, rgba(255,255,255,0.08)'; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
            >
              {/* expand/collapse arrow */}
              {hasChildren ? (
                <span
                  onClick={(e) => { e.stopPropagation(); toggle(node.dn); }}
                  style={{
                    width: 20,
                    height: 20,
                    display: 'inline-flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: 10,
                    color: 'var(--ant-color-text-secondary)',
                    flexShrink: 0,
                  }}
                >
                  {isCollapsed ? <RightOutlined /> : <DownOutlined />}
                </span>
              ) : (
                <span style={{ width: 20, flexShrink: 0 }} />
              )}

              {/* icon */}
              <span style={{ marginRight: 6, color: isContainer ? 'var(--ant-color-primary)' : 'var(--ant-color-text-secondary)', flexShrink: 0 }}>
                {childTypeIcon(node.type)}
              </span>

              {/* label */}
              <span
                onClick={() => {
                  if (isContainer) {
                    onSelectOU(node.dn);
                  } else {
                    onSelectObject(node.dn, node.type);
                  }
                }}
                style={{
                  flex: 1,
                  minWidth: 0,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  fontWeight: isContainer ? 500 : 400,
                }}
              >
                {node.name}
              </span>

              {/* type tag for leaf objects */}
              {!isContainer && (
                <Tag
                  color={childTypeColor(node.type)}
                  style={{ margin: '0 0 0 8px', fontSize: 11, lineHeight: '18px', padding: '0 5px', flexShrink: 0 }}
                >
                  {node.type}
                </Tag>
              )}

              {/* child count for containers */}
              {isContainer && node.childCount != null && node.childCount > 0 && (
                <span style={{ marginLeft: 8, fontSize: 11, color: 'var(--ant-color-text-tertiary)', flexShrink: 0 }}>
                  {node.childCount}
                </span>
              )}
            </div>

            {/* children */}
            {hasChildren && !isCollapsed && (
              <DirectoryTree
                nodes={node.children!}
                depth={depth + 1}
                onSelectOU={onSelectOU}
                onSelectObject={onSelectObject}
              />
            )}
          </div>
        );
      })}
    </>
  );
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export default function OUs() {
  const { isAdmin } = useAuth();
  const actionRef = useRef<ActionType>(null);
  const navigate = useNavigate();

  /* ---- state ---- */
  const [ous, setOUs] = useState<OU[]>([]);
  const [treeMap, setTreeMap] = useState<Record<string, OU[]>>({});
  const [treeContents, setTreeContents] = useState<Record<string, TreeNode[]>>({});
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [search, setSearch] = useState('');
  const [selectedOU, setSelectedOU] = useState<OU | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [createLoading, setCreateLoading] = useState(false);
  const [ouChildren, setOUChildren] = useState<OUChild[]>([]);
  const [childrenLoading, setChildrenLoading] = useState(false);

  const handleCreateOU = useCallback(async () => {
    try {
      const values = await createForm.validateFields();
      setCreateLoading(true);
      await api.post('/ous', {
        name: values.name,
        description: values.description,
        parentDn: values.parentDn,
      });
      notification.success({ message: `OU "${values.name}" created` });
      createForm.resetFields();
      setCreateOpen(false);
      refresh();
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        notification.error({ message: 'Create OU failed', description: err.message });
      }
    } finally {
      setCreateLoading(false);
    }
  }, [createForm]);

  const handleDeleteOU = useCallback(async (ou: OU) => {
    Modal.confirm({
      title: 'Delete Organizational Unit',
      icon: <ExclamationCircleOutlined />,
      content: `Delete OU "${ou.name}"? This cannot be undone.${ou.childCount > 0 ? ` Warning: this OU has ${ou.childCount} child objects.` : ''}`,
      okText: 'Delete',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await api.delete(`/ous/${encodeURIComponent(ou.dn)}`);
          notification.success({ message: `OU "${ou.name}" deleted` });
          setDrawerOpen(false);
          setSelectedOU(null);
          refresh();
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : 'Delete failed';
          notification.error({ message: 'Delete OU failed', description: msg });
        }
      },
    });
  }, []);

  /* ---- load OU contents ---- */
  const loadOUContents = useCallback(async (dn: string) => {
    setChildrenLoading(true);
    try {
      const data = await api.get<{ children: OUChild[] }>(`/ous/${encodeURIComponent(dn)}/contents`);
      setOUChildren(data.children || []);
    } catch {
      setOUChildren([]);
    } finally {
      setChildrenLoading(false);
    }
  }, []);

  /* ---- data loading ---- */
  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ ous: OU[]; total: number }>('/ous');
      setOUs(data.ous);
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }, []);

  const loadFullTree = useCallback(async () => {
    try {
      const data = await api.get<{
        tree: Record<string, OU[]>;
        contents: Record<string, TreeNode[]>;
      }>('/ous/tree/full');
      setTreeMap(data.tree);
      setTreeContents(data.contents || {});
    } catch {
      // API unavailable — fall back to basic tree
      try {
        const data = await api.get<{ tree: Record<string, OU[]> }>('/ous/tree');
        setTreeMap(data.tree);
      } catch {
        // both unavailable
      }
    }
  }, []);

  const refresh = useCallback(() => {
    loadList();
    loadFullTree();
  }, [loadList, loadFullTree]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  /* ---- filtering ---- */
  const filteredOUs = useMemo(() => {
    if (!search) return ous;
    const q = search.toLowerCase();
    return ous.filter(
      (ou) =>
        ou.name.toLowerCase().includes(q) ||
        ou.description.toLowerCase().includes(q),
    );
  }, [ous, search]);

  /* ---- tree data ---- */
  const dirTree = useMemo((): DirNode[] => {
    const rootKeys = Object.keys(treeMap);
    if (rootKeys.length === 0) return [];

    // The root key is the base DN — the one that is a parent but not a child anywhere
    const allChildDNs = new Set(
      Object.values(treeMap).flat().map((ou) => ou.dn),
    );
    const rootKey = rootKeys.find((k) => !allChildDNs.has(k)) || rootKeys[0];

    return buildDirTree(treeMap, treeContents, rootKey);
  }, [treeMap, treeContents]);

  /* ---- OU lookup for tree clicks ---- */
  const ouByDN = useMemo(() => {
    const map = new Map<string, OU>();
    for (const ouList of Object.values(treeMap)) {
      for (const ou of ouList) {
        map.set(ou.dn, ou);
      }
    }
    return map;
  }, [treeMap]);

  const handleTreeSelectOU = useCallback((dn: string) => {
    const ou = ouByDN.get(dn);
    if (ou) {
      setSelectedOU(ou);
      setDrawerOpen(true);
      loadOUContents(ou.dn);
    }
  }, [ouByDN, loadOUContents]);

  const handleTreeSelectObject = useCallback((dn: string, type: string) => {
    const path = objectNavPath(type);
    if (path) navigate(path);
  }, [navigate]);

  /* ---- open drawer ---- */
  const openDetail = (ou: OU) => {
    setSelectedOU(ou);
    setDrawerOpen(true);
    loadOUContents(ou.dn);
  };

  /* ---- ProTable columns ---- */
  const columns: ProColumns<OU>[] = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      render: (_, record) => (
        <Space>
          <FolderOutlined style={{ color: 'var(--ant-color-primary)' }} />
          <a onClick={() => openDetail(record)}>{record.name}</a>
        </Space>
      ),
    },
    {
      title: 'Distinguished Name',
      dataIndex: 'dn',
      key: 'dn',
      ellipsis: true,
      render: (_, record) => (
        <Text style={MONO} ellipsis={{ tooltip: record.dn }}>
          {record.dn}
        </Text>
      ),
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      responsive: ['lg'],
    },
    {
      title: 'Children',
      dataIndex: 'childCount',
      key: 'childCount',
      width: 100,
      align: 'right' as const,
      sorter: (a, b) => a.childCount - b.childCount,
      render: (_, record) =>
        record.childCount > 0 ? (
          <Tag>{record.childCount}</Tag>
        ) : (
          <Text type="secondary">0</Text>
        ),
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      render: (_, record) => {
        const viewItem = { key: 'view', icon: <FolderOpenOutlined />, label: 'View Details' };
        const adminItems = isAdmin ? [
          { key: 'edit', icon: <EditOutlined />, label: 'Edit OU' },
          { type: 'divider' as const },
          { key: 'move', label: 'Move OU' },
          { type: 'divider' as const },
          { key: 'delete', icon: <DeleteOutlined />, label: 'Delete OU', danger: true },
        ] : [];
        return (
          <Dropdown
            menu={{
              items: [viewItem, ...adminItems],
              onClick: ({ key }) => {
                if (key === 'view') {
                  openDetail(record);
                } else if (key === 'delete') {
                  handleDeleteOU(record);
                } else {
                  notification.info({ message: `${key} — not yet implemented` });
                }
              },
            }}
            trigger={['click']}
          >
            <Button type="text" icon={<MoreOutlined />} size="small" />
          </Dropdown>
        );
      },
    },
  ];

  /* ---------------------------------------------------------------- */
  /*  Render                                                           */
  /* ---------------------------------------------------------------- */

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space align="center">
          <ApartmentOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Organizational Units</Title>
          <Text type="secondary">({ous.length})</Text>
        </Space>
        <Space>
          <Segmented
            value={viewMode}
            onChange={(v) => setViewMode(v as ViewMode)}
            options={[
              { label: 'List', value: 'list' },
              { label: 'Tree', value: 'tree' },
            ]}
          />
        </Space>
      </div>

      {/* List view */}
      {viewMode === 'list' && (
        <ProTable<OU>
          actionRef={actionRef}
          columns={columns}
          dataSource={filteredOUs}
          rowKey="dn"
          loading={loading}
          search={false}
          dateFormatter="string"
          options={false}
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            showTotal: (total) => `${total} OUs`,
          }}
          toolBarRender={() => [
            <Input
              key="search"
              placeholder="Search OUs..."
              prefix={<SearchOutlined />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              allowClear
              style={{ width: 240 }}
            />,
            <Button key="refresh" icon={<ReloadOutlined />} onClick={refresh} />,
            ...(isAdmin ? [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setCreateOpen(true)}
              >
                New OU
              </Button>,
            ] : []),
          ]}
        />
      )}

      {/* Tree view */}
      {viewMode === 'tree' && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <Input
              placeholder="Search tree..."
              prefix={<SearchOutlined />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              allowClear
              style={{ width: 280 }}
            />
            <Space>
              <Button icon={<ReloadOutlined />} onClick={refresh} />
              {isAdmin && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setCreateOpen(true)}
                >
                  New OU
                </Button>
              )}
            </Space>
          </div>

          {dirTree.length > 0 ? (
            <div style={{
              border: '1px solid var(--ant-color-border, #303030)',
              borderRadius: 8,
              padding: '8px 4px',
              fontSize: 13,
            }}>
              <DirectoryTree
                nodes={dirTree}
                onSelectOU={handleTreeSelectOU}
                onSelectObject={handleTreeSelectObject}
              />
            </div>
          ) : (
            <div style={{ padding: 48, textAlign: 'center' }}>
              <FolderOutlined style={{ fontSize: 36, color: 'var(--ant-color-text-tertiary)', marginBottom: 12 }} />
              <br />
              <Text type="secondary">
                {search ? 'No items match the current search.' : 'No organizational units found.'}
              </Text>
            </div>
          )}
        </div>
      )}

      {/* Detail Drawer */}
      <Drawer
        title={
          <Space>
            <FolderOutlined />
            <span>{selectedOU?.name}</span>
          </Space>
        }
        placement="right"
        width={520}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedOU(null); }}
        extra={
          <Space>
            <Button
              icon={<EditOutlined />}
              onClick={() => notification.info({ message: 'Edit OU — not yet implemented' })}
            >
              Edit
            </Button>
          </Space>
        }
      >
        {selectedOU && (
          <>
            {/* Contents */}
            <Title level={5} style={{ marginBottom: 12 }}>
              Contents ({childrenLoading ? '...' : ouChildren.length})
            </Title>
            {childrenLoading ? (
              <Text type="secondary">Loading...</Text>
            ) : ouChildren.length > 0 ? (
              <div style={{ maxHeight: 300, overflowY: 'auto', marginBottom: 8 }}>
                <Space direction="vertical" size={2} style={{ width: '100%' }}>
                  {ouChildren.map((child) => (
                    <div key={child.dn} style={{
                      display: 'flex', alignItems: 'center', gap: 8,
                      padding: '4px 8px', borderRadius: 4,
                      border: '1px solid var(--ant-color-border-secondary, #303030)',
                    }}>
                      {childTypeIcon(child.objectClass)}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <Text ellipsis style={{ display: 'block' }}>{child.name}</Text>
                        {child.description && (
                          <Text type="secondary" style={{ fontSize: 11 }} ellipsis>{child.description}</Text>
                        )}
                      </div>
                      <Tag color={childTypeColor(child.objectClass)} style={{ margin: 0, fontSize: 11 }}>
                        {child.objectClass}
                      </Tag>
                    </div>
                  ))}
                </Space>
              </div>
            ) : (
              <Text type="secondary">No objects in this OU</Text>
            )}

            <Divider />

            {/* General */}
            <Title level={5} style={{ marginBottom: 12 }}>General</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Name">{selectedOU.name}</Descriptions.Item>
              <Descriptions.Item label="Description">
                {selectedOU.description || <Text type="secondary">No description</Text>}
              </Descriptions.Item>
            </Descriptions>

            <Divider />

            {/* Location in Directory */}
            <Title level={5} style={{ marginBottom: 12 }}>Location in Directory</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Distinguished Name">
                <Space>
                  <Text
                    style={{ ...MONO, wordBreak: 'break-all' }}
                  >
                    {selectedOU.dn}
                  </Text>
                  <Tooltip title="Copy DN">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => copyToClipboard(selectedOU.dn, 'DN')}
                    />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="CN">
                <Text code>{cnFromDN(selectedOU.dn)}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Parent OU">
                <Space>
                  <Text style={{ ...MONO, wordBreak: 'break-all', fontSize: 11 }}>
                    {parentDN(selectedOU.dn)}
                  </Text>
                  <Tooltip title="Copy parent DN">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => copyToClipboard(parentDN(selectedOU.dn), 'Parent DN')}
                    />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
            </Descriptions>

            <Divider />

            {/* Actions (admin only) */}
            {isAdmin && (
              <Space direction="vertical" style={{ width: '100%' }}>
                <Button
                  block
                  icon={<ApartmentOutlined />}
                  onClick={() => notification.info({ message: 'Move OU — not yet implemented' })}
                >
                  Move to Another OU
                </Button>
                <Button
                  block
                  danger
                  type="primary"
                  icon={<DeleteOutlined />}
                  onClick={() => selectedOU && handleDeleteOU(selectedOU)}
                >
                  Delete Organizational Unit
                </Button>
              </Space>
            )}
          </>
        )}
      </Drawer>

      {/* Create OU Modal */}
      <Modal
        title="Create Organizational Unit"
        open={createOpen}
        onCancel={() => { createForm.resetFields(); setCreateOpen(false); }}
        onOk={handleCreateOU}
        confirmLoading={createLoading}
        okText="Create OU"
      >
        <Form form={createForm} layout="vertical">
          <Form.Item
            name="name"
            label="OU Name"
            rules={[{ required: true, message: 'OU name is required' }]}
          >
            <Input placeholder="e.g. Engineering" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Optional description" />
          </Form.Item>
          <Form.Item name="parentDn" label="Parent OU">
            <Select
              placeholder="Root (Base DN)"
              allowClear
              options={ous.map((ou) => ({ value: ou.dn, label: ou.name }))}
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
