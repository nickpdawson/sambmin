import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Button, Input, Space, Tag, Tabs, Tooltip, Dropdown, Badge, Typography,
  notification,
} from 'antd';
import {
  PlusOutlined, ReloadOutlined,
  MoreOutlined, LockOutlined, StopOutlined,
  CheckCircleOutlined, SearchOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { api } from '../../api/client';
import UserDrawer from './UserDrawer';
import CreateUserDrawer from './CreateUserDrawer';

const { Text } = Typography;

interface User {
  dn: string;
  samAccountName: string;
  displayName: string;
  givenName: string;
  sn: string;
  mail: string;
  userPrincipalName: string;
  department: string;
  title: string;
  enabled: boolean;
  lockedOut: boolean;
  lastLogon: string;
  whenCreated: string;
  memberOf: string[];
}

type TabFilter = 'all' | 'active' | 'disabled' | 'locked';

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export default function Users() {
  const actionRef = useRef<ActionType>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ users: User[] }>('/users');
      setUsers(data.users);
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  const filteredUsers = users.filter((u) => {
    if (tabFilter === 'active' && (!u.enabled || u.lockedOut)) return false;
    if (tabFilter === 'disabled' && u.enabled) return false;
    if (tabFilter === 'locked' && !u.lockedOut) return false;

    if (search) {
      const s = search.toLowerCase();
      return (
        u.displayName.toLowerCase().includes(s) ||
        u.samAccountName.toLowerCase().includes(s) ||
        u.mail.toLowerCase().includes(s) ||
        u.department.toLowerCase().includes(s)
      );
    }
    return true;
  });

  const lockedCount = users.filter((u) => u.lockedOut).length;
  const disabledCount = users.filter((u) => !u.enabled).length;

  const columns: ProColumns<User>[] = [
    {
      title: 'Name',
      dataIndex: 'displayName',
      key: 'displayName',
      sorter: (a, b) => a.displayName.localeCompare(b.displayName),
      render: (_, record) => (
        <div>
          <a onClick={() => { setSelectedUser(record); setDrawerOpen(true); }}>
            {record.displayName}
          </a>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>
            {record.samAccountName}
          </Text>
        </div>
      ),
    },
    {
      title: 'Email',
      dataIndex: 'mail',
      key: 'mail',
      copyable: true,
      ellipsis: true,
      responsive: ['lg'],
    },
    {
      title: 'Department',
      dataIndex: 'department',
      key: 'department',
      filters: [...new Set(users.map((u) => u.department))].map((d) => ({ text: d, value: d })),
      onFilter: (value, record) => record.department === value,
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
      responsive: ['xl'],
      ellipsis: true,
    },
    {
      title: 'Status',
      key: 'status',
      width: 120,
      render: (_, record) => {
        if (record.lockedOut) return <Tag icon={<LockOutlined />} color="error">Locked</Tag>;
        if (!record.enabled) return <Tag icon={<StopOutlined />} color="default">Disabled</Tag>;
        return <Tag icon={<CheckCircleOutlined />} color="success">Active</Tag>;
      },
    },
    {
      title: 'Last Logon',
      dataIndex: 'lastLogon',
      key: 'lastLogon',
      width: 120,
      responsive: ['lg'],
      sorter: (a, b) => new Date(a.lastLogon).getTime() - new Date(b.lastLogon).getTime(),
      render: (_, record) => (
        <Tooltip title={new Date(record.lastLogon).toLocaleString()}>
          <Text type="secondary" style={{ fontSize: 13 }}>{timeAgo(record.lastLogon)}</Text>
        </Tooltip>
      ),
    },
    {
      title: 'Groups',
      key: 'groups',
      responsive: ['xl'],
      render: (_, record) => (
        <Space size={4} wrap>
          {record.memberOf.slice(0, 2).map((g) => (
            <Tag key={g} style={{ fontSize: 11 }}>{g}</Tag>
          ))}
          {record.memberOf.length > 2 && (
            <Tag style={{ fontSize: 11 }}>+{record.memberOf.length - 2}</Tag>
          )}
        </Space>
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
              { key: 'view', label: 'View Details' },
              { key: 'reset', label: 'Reset Password' },
              { type: 'divider' },
              ...(record.lockedOut ? [{ key: 'unlock', label: 'Unlock Account' }] : []),
              record.enabled
                ? { key: 'disable', label: 'Disable Account', danger: true }
                : { key: 'enable', label: 'Enable Account' },
              { type: 'divider' },
              { key: 'delete', label: 'Delete User', danger: true },
            ],
            onClick: ({ key }) => {
              if (key === 'view') {
                setSelectedUser(record);
                setDrawerOpen(true);
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

  const tabItems = [
    { key: 'all', label: `All Users (${users.length})` },
    { key: 'active', label: `Active (${users.length - disabledCount - lockedCount})` },
    { key: 'disabled', label: <Badge count={disabledCount} size="small" offset={[8, 0]}>Disabled</Badge> },
    { key: 'locked', label: <Badge count={lockedCount} size="small" offset={[8, 0]} color="red">Locked Out</Badge> },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Tabs
        activeKey={tabFilter}
        onChange={(key) => setTabFilter(key as TabFilter)}
        items={tabItems}
        style={{ marginBottom: -8 }}
      />

      <ProTable<User>
        actionRef={actionRef}
        columns={columns}
        dataSource={filteredUsers}
        rowKey="dn"
        loading={loading}
        search={false}
        dateFormatter="string"
        options={false}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `${total} users`,
        }}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as string[]),
        }}
        toolBarRender={() => [
          <Input
            key="search"
            placeholder="Search users..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            allowClear
            style={{ width: 240 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={loadUsers} />,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            New User
          </Button>,
        ]}
        headerTitle={
          selectedRowKeys.length > 0 ? (
            <Space>
              <Text>{selectedRowKeys.length} selected</Text>
              <Button size="small" onClick={() => notification.info({ message: 'Bulk enable — not yet implemented' })}>Enable</Button>
              <Button size="small" onClick={() => notification.info({ message: 'Bulk disable — not yet implemented' })}>Disable</Button>
              <Button size="small" danger onClick={() => notification.info({ message: 'Bulk delete — not yet implemented' })}>Delete</Button>
              <Button size="small" type="link" onClick={() => setSelectedRowKeys([])}>Clear</Button>
            </Space>
          ) : undefined
        }
      />

      <UserDrawer
        user={selectedUser}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedUser(null); }}
      />

      <CreateUserDrawer
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={() => { setCreateOpen(false); loadUsers(); }}
      />
    </Space>
  );
}
