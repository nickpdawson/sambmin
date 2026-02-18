import { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import {
  Button, Input, Space, Typography, Drawer, Descriptions, Divider,
  Tooltip, Dropdown, Segmented, Tree, notification, Tag, Badge,
  Modal, Form, Select,
} from 'antd';
import {
  ApartmentOutlined, FolderOutlined, CopyOutlined,
  PlusOutlined, ReloadOutlined, SearchOutlined, MoreOutlined,
  DeleteOutlined, EditOutlined, FolderOpenOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import type { DataNode } from 'antd/es/tree';
import { api } from '../../api/client';

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

type ViewMode = 'list' | 'tree';

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

/* ------------------------------------------------------------------ */
/*  Tree builder — converts { parentDN: OU[] } map to DataNode[]      */
/* ------------------------------------------------------------------ */

function buildTreeData(
  treeMap: Record<string, OU[]>,
  parentKey: string,
  onSelect: (ou: OU) => void,
): DataNode[] {
  const children = treeMap[parentKey];
  if (!children || children.length === 0) return [];

  return children.map((ou) => ({
    key: ou.dn,
    icon: <FolderOutlined />,
    title: (
      <Space size={8}>
        <a
          onClick={(e) => { e.stopPropagation(); onSelect(ou); }}
          style={{ fontWeight: 500 }}
        >
          {ou.name}
        </a>
        {ou.childCount > 0 && (
          <Badge
            count={ou.childCount}
            size="small"
            style={{ backgroundColor: 'var(--ant-color-primary)' }}
          />
        )}
        <Text type="secondary" style={{ ...MONO, fontSize: 11 }} ellipsis>
          {ou.dn}
        </Text>
      </Space>
    ),
    children: buildTreeData(treeMap, ou.dn, onSelect),
  }));
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export default function OUs() {
  const actionRef = useRef<ActionType>(null);

  /* ---- state ---- */
  const [ous, setOUs] = useState<OU[]>([]);
  const [treeMap, setTreeMap] = useState<Record<string, OU[]>>({});
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [search, setSearch] = useState('');
  const [selectedOU, setSelectedOU] = useState<OU | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [createLoading, setCreateLoading] = useState(false);

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

  const loadTree = useCallback(async () => {
    try {
      const data = await api.get<{ tree: Record<string, OU[]> }>('/ous/tree');
      setTreeMap(data.tree);
    } catch {
      // API unavailable
    }
  }, []);

  const refresh = useCallback(() => {
    loadList();
    loadTree();
  }, [loadList, loadTree]);

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
  const treeData = useMemo(() => {
    const rootKeys = Object.keys(treeMap);
    if (rootKeys.length === 0) return [];

    const handleSelect = (ou: OU) => {
      setSelectedOU(ou);
      setDrawerOpen(true);
    };

    // The root key is the base DN — the one that is a parent but not a child anywhere
    const allChildDNs = new Set(
      Object.values(treeMap).flat().map((ou) => ou.dn),
    );
    const rootKey = rootKeys.find((k) => !allChildDNs.has(k)) || rootKeys[0];

    return buildTreeData(treeMap, rootKey, handleSelect);
  }, [treeMap]);

  /* ---- filtered tree (when searching) ---- */
  const filteredTreeData = useMemo(() => {
    if (!search) return treeData;

    const q = search.toLowerCase();

    function filterNodes(nodes: DataNode[]): DataNode[] {
      return nodes.reduce<DataNode[]>((acc, node) => {
        const ou = ous.find((o) => o.dn === node.key);
        const matchesSelf =
          ou &&
          (ou.name.toLowerCase().includes(q) ||
            ou.description.toLowerCase().includes(q));
        const filteredChildren = node.children
          ? filterNodes(node.children as DataNode[])
          : [];

        if (matchesSelf || filteredChildren.length > 0) {
          acc.push({ ...node, children: filteredChildren });
        }
        return acc;
      }, []);
    }

    return filterNodes(treeData);
  }, [treeData, search, ous]);

  /* ---- open drawer ---- */
  const openDetail = (ou: OU) => {
    setSelectedOU(ou);
    setDrawerOpen(true);
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
      render: (_, record) => (
        <Dropdown
          menu={{
            items: [
              { key: 'view', icon: <FolderOpenOutlined />, label: 'View Details' },
              { key: 'edit', icon: <EditOutlined />, label: 'Edit OU' },
              { type: 'divider' },
              { key: 'move', label: 'Move OU' },
              { type: 'divider' },
              { key: 'delete', icon: <DeleteOutlined />, label: 'Delete OU', danger: true },
            ],
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
      ),
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
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateOpen(true)}
            >
              New OU
            </Button>,
          ]}
        />
      )}

      {/* Tree view */}
      {viewMode === 'tree' && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <Input
              placeholder="Search OUs..."
              prefix={<SearchOutlined />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              allowClear
              style={{ width: 280 }}
            />
            <Space>
              <Button icon={<ReloadOutlined />} onClick={refresh} />
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setCreateOpen(true)}
              >
                New OU
              </Button>
            </Space>
          </div>

          {filteredTreeData.length > 0 ? (
            <Tree
              showIcon
              defaultExpandAll
              treeData={filteredTreeData}
              blockNode
              style={{ padding: '8px 0' }}
            />
          ) : (
            <div style={{ padding: 48, textAlign: 'center' }}>
              <FolderOutlined style={{ fontSize: 36, color: 'var(--ant-color-text-tertiary)', marginBottom: 12 }} />
              <br />
              <Text type="secondary">
                {search ? 'No OUs match the current search.' : 'No organizational units found.'}
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
            {/* General */}
            <Title level={5} style={{ marginBottom: 12 }}>General</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Name">{selectedOU.name}</Descriptions.Item>
              <Descriptions.Item label="Description">
                {selectedOU.description || <Text type="secondary">No description</Text>}
              </Descriptions.Item>
              <Descriptions.Item label="Child OUs">
                {selectedOU.childCount > 0 ? (
                  <Tag>{selectedOU.childCount}</Tag>
                ) : (
                  <Text type="secondary">None</Text>
                )}
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

            {/* Actions */}
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
