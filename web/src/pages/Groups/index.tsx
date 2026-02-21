import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Button, Input, Space, Tag, Tabs, Tooltip, Typography, Drawer, Descriptions,
  Divider, List, notification, Dropdown, Modal, Form, Select,
} from 'antd';
import {
  PlusOutlined, ReloadOutlined, SearchOutlined, TeamOutlined,
  SafetyCertificateOutlined, MailOutlined, CopyOutlined, MoreOutlined,
  ExclamationCircleOutlined, DeleteOutlined, UserOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import ExportButton from '../../components/ExportButton';
import CreateGroupDrawer from './CreateGroupDrawer';

const { Text, Title } = Typography;

interface Group {
  dn: string;
  name: string;
  samAccountName: string;
  description: string;
  groupType: string;
  groupScope: string;
  members: string[];
  memberOf: string[];
}

type TabFilter = 'all' | 'security' | 'distribution';

const cnFromDN = (dn: string) => dn.split(',')[0]?.replace(/^CN=/i, '') || dn;

const monoStyle: React.CSSProperties = { fontFamily: '"JetBrains Mono", monospace' };

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

function scopeColor(scope: string): string {
  switch (scope) {
    case 'global': return 'blue';
    case 'domainLocal': return 'orange';
    case 'universal': return 'purple';
    default: return 'default';
  }
}

function scopeLabel(scope: string): string {
  switch (scope) {
    case 'global': return 'Global';
    case 'domainLocal': return 'Domain Local';
    case 'universal': return 'Universal';
    default: return scope;
  }
}

export default function Groups() {
  const { isAdmin } = useAuth();
  const actionRef = useRef<ActionType>(null);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<Group | null>(null);
  const [renameForm] = Form.useForm();
  const [addMemberOpen, setAddMemberOpen] = useState(false);
  const [selectedMemberDn, setSelectedMemberDn] = useState<string | null>(null);
  const [addingMember, setAddingMember] = useState(false);
  const [allUsers, setAllUsers] = useState<{ dn: string; displayName: string; samAccountName: string }[]>([]);

  const loadGroups = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ groups: Group[]; total: number }>('/groups');
      setGroups(data.groups);
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }, []);

  const handleGroupAction = useCallback(async (action: string, record: Group) => {
    const dn = encodeURIComponent(record.dn);

    if (action === 'delete') {
      Modal.confirm({
        title: 'Delete Group',
        icon: <ExclamationCircleOutlined />,
        content: `Are you sure you want to delete ${record.name}? This cannot be undone.`,
        okText: 'Delete',
        okButtonProps: { danger: true },
        onOk: async () => {
          await api.delete(`/groups/${dn}`);
          notification.success({ message: `${record.name} deleted` });
          loadGroups();
        },
      });
      return;
    }

    if (action === 'rename') {
      setRenameTarget(record);
      renameForm.setFieldsValue({ newName: record.name });
      return;
    }
  }, [loadGroups, renameForm]);

  const handleRenameGroup = useCallback(async () => {
    if (!renameTarget) return;
    try {
      const values = await renameForm.validateFields();
      const dn = encodeURIComponent(renameTarget.dn);
      await api.post(`/groups/${dn}/rename`, { newName: values.newName });
      notification.success({ message: `Renamed to ${values.newName}` });
      renameForm.resetFields();
      setRenameTarget(null);
      loadGroups();
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        Modal.error({ title: 'Rename failed', content: err.message });
      }
    }
  }, [renameTarget, renameForm, loadGroups]);

  const refreshSelectedGroup = useCallback(async (groupDn: string) => {
    try {
      const fresh = await api.get<Group>(`/groups/${encodeURIComponent(groupDn)}`);
      setSelectedGroup(fresh);
    } catch { /* keep stale data */ }
  }, []);

  const handleAddMember = useCallback(async () => {
    if (!selectedGroup || !selectedMemberDn) return;
    setAddingMember(true);
    try {
      const groupDn = encodeURIComponent(selectedGroup.dn);
      await api.post(`/groups/${groupDn}/members`, { memberDn: selectedMemberDn });
      notification.success({ message: `Member added to ${selectedGroup.name}` });
      setAddMemberOpen(false);
      setSelectedMemberDn(null);
      await refreshSelectedGroup(selectedGroup.dn);
      loadGroups();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to add member';
      Modal.error({ title: 'Add member failed', content: msg });
    } finally {
      setAddingMember(false);
    }
  }, [selectedGroup, selectedMemberDn, refreshSelectedGroup, loadGroups]);

  const handleRemoveMember = useCallback(async (memberDN: string) => {
    if (!selectedGroup) return;
    Modal.confirm({
      title: 'Remove Member',
      icon: <ExclamationCircleOutlined />,
      content: `Remove ${cnFromDN(memberDN)} from ${selectedGroup.name}?`,
      okText: 'Remove',
      okButtonProps: { danger: true },
      onOk: async () => {
        const gdn = encodeURIComponent(selectedGroup.dn);
        const mdn = encodeURIComponent(memberDN);
        await api.delete(`/groups/${gdn}/members/${mdn}`);
        notification.success({ message: `${cnFromDN(memberDN)} removed` });
        await refreshSelectedGroup(selectedGroup.dn);
        loadGroups();
      },
    });
  }, [selectedGroup, refreshSelectedGroup, loadGroups]);

  // Load users when add member modal opens
  useEffect(() => {
    if (addMemberOpen && allUsers.length === 0) {
      api.get<{ users: { dn: string; displayName: string; samAccountName: string }[] }>('/users')
        .then(data => setAllUsers(data.users))
        .catch(() => {});
    }
  }, [addMemberOpen, allUsers.length]);

  useEffect(() => {
    loadGroups();
  }, [loadGroups]);

  const filteredGroups = groups.filter((g) => {
    if (tabFilter === 'security' && g.groupType !== 'security') return false;
    if (tabFilter === 'distribution' && g.groupType !== 'distribution') return false;

    if (search) {
      const s = search.toLowerCase();
      return (
        g.name.toLowerCase().includes(s) ||
        g.samAccountName.toLowerCase().includes(s) ||
        g.description.toLowerCase().includes(s)
      );
    }
    return true;
  });

  const securityCount = groups.filter((g) => g.groupType === 'security').length;
  const distributionCount = groups.filter((g) => g.groupType === 'distribution').length;

  const columns: ProColumns<Group>[] = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      render: (_, record) => (
        <div>
          <a onClick={() => { setSelectedGroup(record); setDrawerOpen(true); }}>
            {record.name}
          </a>
          <br />
          <Text type="secondary" style={{ fontSize: 12, ...monoStyle }}>
            {record.samAccountName}
          </Text>
        </div>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'groupType',
      key: 'groupType',
      width: 130,
      render: (_, record) =>
        record.groupType === 'security' ? (
          <Tag icon={<SafetyCertificateOutlined />} color="green">Security</Tag>
        ) : (
          <Tag icon={<MailOutlined />} color="blue">Distribution</Tag>
        ),
    },
    {
      title: 'Scope',
      dataIndex: 'groupScope',
      key: 'groupScope',
      width: 140,
      filters: [
        { text: 'Global', value: 'global' },
        { text: 'Domain Local', value: 'domainLocal' },
        { text: 'Universal', value: 'universal' },
      ],
      onFilter: (value, record) => record.groupScope === value,
      render: (_, record) => (
        <Tag color={scopeColor(record.groupScope)}>
          {scopeLabel(record.groupScope)}
        </Tag>
      ),
    },
    {
      title: 'Members',
      key: 'members',
      width: 100,
      sorter: (a, b) => a.members.length - b.members.length,
      render: (_, record) => (
        <Space size={4}>
          <TeamOutlined />
          <span>{record.members.length}</span>
        </Space>
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
      title: '',
      key: 'actions',
      width: 48,
      render: (_, record) => {
        const viewItem = { key: 'view', label: 'View Details' };
        const adminItems = isAdmin ? [
          { key: 'rename', label: 'Rename' },
          { type: 'divider' as const },
          { key: 'delete', label: 'Delete Group', danger: true },
        ] : [];
        return (
          <Dropdown
            menu={{
              items: [viewItem, ...adminItems],
              onClick: ({ key }) => {
                if (key === 'view') {
                  setSelectedGroup(record);
                  setDrawerOpen(true);
                } else {
                  handleGroupAction(key, record);
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

  const tabItems = [
    { key: 'all', label: `All Groups (${groups.length})` },
    { key: 'security', label: `Security (${securityCount})` },
    { key: 'distribution', label: `Distribution (${distributionCount})` },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Tabs
        activeKey={tabFilter}
        onChange={(key) => setTabFilter(key as TabFilter)}
        items={tabItems}
        style={{ marginBottom: -8 }}
      />

      <ProTable<Group>
        actionRef={actionRef}
        columns={columns}
        dataSource={filteredGroups}
        rowKey="dn"
        loading={loading}
        search={false}
        dateFormatter="string"
        options={false}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `${total} groups`,
        }}
        toolBarRender={() => [
          <Input
            key="search"
            placeholder="Search groups..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            allowClear
            style={{ width: 240 }}
          />,
          <ExportButton
            key="export"
            data={filteredGroups as unknown as Record<string, unknown>[]}
            filename="sambmin-groups"
            columns={[
              { key: 'name', title: 'Name' },
              { key: 'groupType', title: 'Type' },
              { key: 'groupScope', title: 'Scope' },
              { key: 'description', title: 'Description' },
              { key: 'members', title: 'Members' },
              { key: 'dn', title: 'DN' },
            ]}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={loadGroups} />,
          ...(isAdmin ? [
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateOpen(true)}
            >
              New Group
            </Button>,
          ] : []),
        ]}
      />

      {/* Group Detail Drawer */}
      <Drawer
        title={
          selectedGroup ? (
            <Space>
              <TeamOutlined />
              <span>{selectedGroup.name}</span>
              {selectedGroup.groupType === 'security' ? (
                <Tag icon={<SafetyCertificateOutlined />} color="green">Security</Tag>
              ) : (
                <Tag icon={<MailOutlined />} color="blue">Distribution</Tag>
              )}
              <Tag color={scopeColor(selectedGroup.groupScope)}>
                {scopeLabel(selectedGroup.groupScope)}
              </Tag>
            </Space>
          ) : null
        }
        placement="right"
        width={560}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedGroup(null); }}
      >
        {selectedGroup && (
          <>
            {/* Identity */}
            <Title level={5} style={{ marginBottom: 12 }}>Identity</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Name">{selectedGroup.name}</Descriptions.Item>
              <Descriptions.Item label="sAMAccountName">
                <Space>
                  <Text code style={monoStyle}>{selectedGroup.samAccountName}</Text>
                  <Tooltip title="Copy sAMAccountName">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => copyToClipboard(selectedGroup.samAccountName, 'sAMAccountName')}
                    />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="DN">
                <Space>
                  <Text
                    code
                    style={{ fontSize: 12, wordBreak: 'break-all', ...monoStyle }}
                  >
                    {selectedGroup.dn}
                  </Text>
                  <Tooltip title="Copy DN">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => copyToClipboard(selectedGroup.dn, 'DN')}
                    />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
            </Descriptions>

            <Divider />

            {/* Description */}
            <Title level={5} style={{ marginBottom: 12 }}>Description</Title>
            <Text type={selectedGroup.description ? undefined : 'secondary'}>
              {selectedGroup.description || 'No description'}
            </Text>

            <Divider />

            {/* Type & Scope */}
            <Title level={5} style={{ marginBottom: 12 }}>Type & Scope</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Type">
                {selectedGroup.groupType === 'security' ? (
                  <Tag icon={<SafetyCertificateOutlined />} color="green">Security</Tag>
                ) : (
                  <Tag icon={<MailOutlined />} color="blue">Distribution</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="Scope">
                <Tag color={scopeColor(selectedGroup.groupScope)}>
                  {scopeLabel(selectedGroup.groupScope)}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Divider />

            {/* Members */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <Title level={5} style={{ margin: 0 }}>
                Members ({selectedGroup.members.length})
              </Title>
              {isAdmin && (
                <Button size="small" icon={<PlusOutlined />} onClick={() => setAddMemberOpen(true)}>
                  Add Member
                </Button>
              )}
            </div>
            {selectedGroup.members.length > 0 ? (
              <List
                size="small"
                bordered
                dataSource={selectedGroup.members}
                renderItem={(memberDN) => (
                  <List.Item
                    actions={isAdmin ? [
                      <Tooltip key="remove" title="Remove from group">
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => handleRemoveMember(memberDN)}
                        />
                      </Tooltip>,
                    ] : undefined}
                  >
                    <Space direction="vertical" size={0} style={{ width: '100%' }}>
                      <Text><UserOutlined style={{ marginRight: 6 }} />{cnFromDN(memberDN)}</Text>
                      <Text type="secondary" style={{ fontSize: 11, ...monoStyle }}>
                        {memberDN}
                      </Text>
                    </Space>
                  </List.Item>
                )}
              />
            ) : (
              <Text type="secondary">No members</Text>
            )}

            <Divider />

            {/* Member Of */}
            <Title level={5} style={{ marginBottom: 12 }}>
              Member Of ({selectedGroup.memberOf.length})
            </Title>
            {selectedGroup.memberOf.length > 0 ? (
              <Space size={[4, 8]} wrap>
                {selectedGroup.memberOf.map((parentDN) => (
                  <Tag key={parentDN} style={{ cursor: 'default' }}>
                    {cnFromDN(parentDN)}
                  </Tag>
                ))}
              </Space>
            ) : (
              <Text type="secondary">Not a member of any groups</Text>
            )}
          </>
        )}
      </Drawer>

      <CreateGroupDrawer
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={() => { setCreateOpen(false); loadGroups(); }}
      />

      {/* Rename Group Modal */}
      <Modal
        title={`Rename Group — ${renameTarget?.name || ''}`}
        open={!!renameTarget}
        onCancel={() => { renameForm.resetFields(); setRenameTarget(null); }}
        onOk={handleRenameGroup}
        okText="Rename"
      >
        <Form form={renameForm} layout="vertical">
          <Form.Item name="newName" label="New Name" rules={[{ required: true, message: 'New name is required' }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      {/* Add Member Modal */}
      <Modal
        title={`Add Member to ${selectedGroup?.name || ''}`}
        open={addMemberOpen}
        onCancel={() => { setAddMemberOpen(false); setSelectedMemberDn(null); }}
        onOk={handleAddMember}
        okText="Add"
        confirmLoading={addingMember}
        okButtonProps={{ disabled: !selectedMemberDn }}
      >
        <Select
          showSearch
          placeholder="Search by name or username..."
          style={{ width: '100%' }}
          value={selectedMemberDn}
          onChange={(v) => setSelectedMemberDn(v)}
          filterOption={(input, option) => {
            const s = input.toLowerCase();
            const label = (option?.label ?? '').toLowerCase();
            const desc = ((option as any)?.desc ?? '').toLowerCase();
            return label.includes(s) || desc.includes(s);
          }}
          optionRender={(option) => (
            <Space direction="vertical" size={0}>
              <span>{(option as any).data?.label}</span>
              <Text type="secondary" style={{ fontSize: 11, ...monoStyle }}>{(option as any).data?.desc}</Text>
            </Space>
          )}
          options={(allUsers || [])
            .filter(u => !selectedGroup?.members.includes(u.dn))
            .map(u => ({
              label: u.displayName || u.samAccountName,
              value: u.dn,
              desc: u.samAccountName,
            }))}
        />
      </Modal>
    </Space>
  );
}
