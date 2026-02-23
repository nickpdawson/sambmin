import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Button, Input, Space, Tag, Tabs, Tooltip, Dropdown, Badge, Typography,
  notification, Modal, Form,
} from 'antd';
import {
  PlusOutlined, ReloadOutlined,
  MoreOutlined, LockOutlined, StopOutlined,
  CheckCircleOutlined, SearchOutlined, ExclamationCircleOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import ExportButton from '../../components/ExportButton';
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
  description: string;
  department: string;
  title: string;
  company: string;
  manager: string;
  office: string;
  streetAddress: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
  phone: string;
  mobile: string;
  enabled: boolean;
  lockedOut: boolean;
  passwordExpired: boolean;
  accountExpires: string;
  pwdLastSet: string;
  badPwdCount: number;
  lastLogon: string;
  whenCreated: string;
  whenChanged: string;
  memberOf: string[];
}

type TabFilter = 'all' | 'active' | 'disabled' | 'locked';

function timeAgo(iso: string): string {
  if (!iso) return 'Never';
  const d = new Date(iso);
  if (d.getFullYear() < 1971) return 'Never'; // epoch zero / AD "never logged in"
  const diff = Date.now() - d.getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 365) return `${days}d ago`;
  return `${Math.floor(days / 365)}y ago`;
}

export default function Users() {
  const { isAdmin } = useAuth();
  const actionRef = useRef<ActionType>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [resetTarget, setResetTarget] = useState<User | null>(null);
  const [resetForm] = Form.useForm();
  const [renameTarget, setRenameTarget] = useState<User | null>(null);
  const [renameForm] = Form.useForm();
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

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

  const handleUserAction = useCallback(async (action: string, record: User) => {
    const dn = encodeURIComponent(record.dn);
    const name = record.displayName || record.samAccountName;

    if (action === 'delete') {
      setDeleteTarget(record);
      return;
    }

    if (action === 'reset') {
      setResetTarget(record);
      return;
    }

    if (action === 'rename') {
      setRenameTarget(record);
      renameForm.setFieldsValue({ newName: record.displayName || record.samAccountName });
      return;
    }

    try {
      switch (action) {
        case 'enable':
          await api.post(`/users/${dn}/enable`);
          notification.success({ message: `${name} enabled` });
          break;
        case 'disable':
          await api.post(`/users/${dn}/disable`);
          notification.success({ message: `${name} disabled` });
          break;
        case 'unlock':
          await api.post(`/users/${dn}/unlock`);
          notification.success({ message: `${name} unlocked` });
          break;
      }
      loadUsers();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Operation failed';
      Modal.error({ title: `Failed to ${action} user`, content: msg });
    }
  }, [loadUsers]);

  const handleResetPassword = useCallback(async () => {
    if (!resetTarget) return;
    try {
      const values = await resetForm.validateFields();
      const dn = encodeURIComponent(resetTarget.dn);
      await api.post(`/users/${dn}/reset-password`, {
        password: values.password,
        mustChangeAtNextLogin: values.mustChange ?? true,
      });
      notification.success({ message: `Password reset for ${resetTarget.displayName || resetTarget.samAccountName}` });
      resetForm.resetFields();
      setResetTarget(null);
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        Modal.error({ title: 'Password reset failed', content: err.message });
      }
    }
  }, [resetTarget, resetForm]);

  const handleRenameUser = useCallback(async () => {
    if (!renameTarget) return;
    try {
      const values = await renameForm.validateFields();
      const dn = encodeURIComponent(renameTarget.dn);
      await api.post(`/users/${dn}/rename`, { newName: values.newName });
      notification.success({ message: `Renamed to ${values.newName}` });
      renameForm.resetFields();
      setRenameTarget(null);
      loadUsers();
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        Modal.error({ title: 'Rename failed', content: err.message });
      }
    }
  }, [renameTarget, renameForm, loadUsers]);

  const handleDeleteUser = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleteLoading(true);
    try {
      const dn = encodeURIComponent(deleteTarget.dn);
      await api.delete(`/users/${dn}`);
      notification.success({ message: `${deleteTarget.displayName || deleteTarget.samAccountName} deleted` });
      setDeleteTarget(null);
      loadUsers();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Delete failed';
      notification.error({ message: 'Delete failed', description: msg });
    } finally {
      setDeleteLoading(false);
    }
  }, [deleteTarget, loadUsers]);

  const filteredUsers = users.filter((u) => {
    if (tabFilter === 'active' && (!u.enabled || u.lockedOut)) return false;
    if (tabFilter === 'disabled' && u.enabled) return false;
    if (tabFilter === 'locked' && !u.lockedOut) return false;

    if (search) {
      const s = search.toLowerCase();
      return (
        (u.displayName || '').toLowerCase().includes(s) ||
        (u.samAccountName || '').toLowerCase().includes(s) ||
        (u.mail || '').toLowerCase().includes(s) ||
        (u.department || '').toLowerCase().includes(s)
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
      sorter: (a, b) => (a.displayName || '').localeCompare(b.displayName || ''),
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
      filters: [...new Set(users.map((u) => u.department || ''))].filter(Boolean).map((d) => ({ text: d, value: d })),
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
          {(record.memberOf || []).slice(0, 2).map((g) => (
            <Tag key={g} style={{ fontSize: 11 }}>{g}</Tag>
          ))}
          {(record.memberOf || []).length > 2 && (
            <Tag style={{ fontSize: 11 }}>+{record.memberOf.length - 2}</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      render: (_, record) => {
        const viewItem = { key: 'view', label: 'View Details' };
        const adminItems = isAdmin ? [
          { key: 'edit', label: 'Edit' },
          { key: 'rename', label: 'Rename' },
          { key: 'reset', label: 'Reset Password' },
          { type: 'divider' as const },
          ...(record.lockedOut ? [{ key: 'unlock', label: 'Unlock Account' }] : []),
          record.enabled
            ? { key: 'disable', label: 'Disable Account', danger: true }
            : { key: 'enable', label: 'Enable Account' },
          { type: 'divider' as const },
          { key: 'delete', label: 'Delete User', danger: true },
        ] : [];
        return (
          <Dropdown
            menu={{
              items: [viewItem, ...adminItems],
              onClick: ({ key }) => {
                if (key === 'view' || key === 'edit') {
                  setSelectedUser(record);
                  setDrawerOpen(true);
                } else {
                  handleUserAction(key, record);
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
        rowSelection={isAdmin ? {
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as string[]),
        } : undefined}
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
          <ExportButton
            key="export"
            data={filteredUsers as unknown as Record<string, unknown>[]}
            filename="sambmin-users"
            columns={[
              { key: 'samAccountName', title: 'Username' },
              { key: 'displayName', title: 'Display Name' },
              { key: 'mail', title: 'Email' },
              { key: 'department', title: 'Department' },
              { key: 'title', title: 'Title' },
              { key: 'enabled', title: 'Enabled' },
              { key: 'lastLogon', title: 'Last Logon' },
              { key: 'dn', title: 'DN' },
            ]}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={loadUsers} />,
          ...(isAdmin ? [
            <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
              New User
            </Button>,
          ] : []),
        ]}
        headerTitle={
          selectedRowKeys.length > 0 ? (
            <Space>
              <Text>{selectedRowKeys.length} selected</Text>
              <Button size="small" onClick={async () => {
                let ok = 0, fail = 0;
                for (const dn of selectedRowKeys) {
                  try { await api.post(`/users/${encodeURIComponent(dn)}/enable`); ok++; } catch { fail++; }
                }
                if (ok > 0) notification.success({ message: `${ok} user(s) enabled` });
                if (fail > 0) notification.error({ message: `${fail} user(s) failed to enable` });
                setSelectedRowKeys([]); loadUsers();
              }}>Enable</Button>
              <Button size="small" onClick={async () => {
                let ok = 0, fail = 0;
                for (const dn of selectedRowKeys) {
                  try { await api.post(`/users/${encodeURIComponent(dn)}/disable`); ok++; } catch { fail++; }
                }
                if (ok > 0) notification.success({ message: `${ok} user(s) disabled` });
                if (fail > 0) notification.error({ message: `${fail} user(s) failed to disable` });
                setSelectedRowKeys([]); loadUsers();
              }}>Disable</Button>
              <Button size="small" danger onClick={() => {
                Modal.confirm({
                  title: 'Delete Users',
                  icon: <ExclamationCircleOutlined />,
                  content: `Delete ${selectedRowKeys.length} selected user(s)? This cannot be undone.`,
                  okText: 'Delete All',
                  okButtonProps: { danger: true },
                  onOk: async () => {
                    let ok = 0, fail = 0;
                    for (const dn of selectedRowKeys) {
                      try { await api.delete(`/users/${encodeURIComponent(dn)}`); ok++; } catch { fail++; }
                    }
                    if (ok > 0) notification.success({ message: `${ok} user(s) deleted` });
                    if (fail > 0) notification.error({ message: `${fail} user(s) failed to delete` });
                    setSelectedRowKeys([]); loadUsers();
                  },
                });
              }}>Delete</Button>
              <Button size="small" type="link" onClick={() => setSelectedRowKeys([])}>Clear</Button>
            </Space>
          ) : undefined
        }
      />

      <UserDrawer
        user={selectedUser}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedUser(null); }}
        onRefresh={async () => {
          await loadUsers();
          // Refresh the selected user with fresh data
          if (selectedUser) {
            try {
              const dn = encodeURIComponent(selectedUser.dn);
              const fresh = await api.get<User>(`/users/${dn}`);
              setSelectedUser(fresh);
            } catch { /* keep stale data */ }
          }
        }}
      />

      <CreateUserDrawer
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={() => { setCreateOpen(false); loadUsers(); }}
      />

      {/* Reset Password Modal */}
      <Modal
        title={`Reset Password — ${resetTarget?.displayName || resetTarget?.samAccountName || ''}`}
        open={!!resetTarget}
        onCancel={() => { resetForm.resetFields(); setResetTarget(null); }}
        onOk={handleResetPassword}
        okText="Reset Password"
      >
        <Form form={resetForm} layout="vertical" initialValues={{ mustChange: true }}>
          <Form.Item
            name="password"
            label="New Password"
            rules={[
              { required: true, message: 'Password is required' },
              { min: 12, message: 'Must be at least 12 characters' },
            ]}
          >
            <Input.Password placeholder="Minimum 12 characters" />
          </Form.Item>
          <Form.Item name="mustChange" valuePropName="checked">
            <input type="checkbox" /> Must change at next login
          </Form.Item>
        </Form>
      </Modal>

      {/* Rename User Modal */}
      <Modal
        title={`Rename User — ${renameTarget?.displayName || renameTarget?.samAccountName || ''}`}
        open={!!renameTarget}
        onCancel={() => { renameForm.resetFields(); setRenameTarget(null); }}
        onOk={handleRenameUser}
        okText="Rename"
      >
        <Form form={renameForm} layout="vertical">
          <Form.Item name="newName" label="New Name (CN)" rules={[{ required: true, message: 'New name is required' }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      {/* Delete User Confirmation Modal */}
      <Modal
        title="Delete User"
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(null)}
        onOk={handleDeleteUser}
        okText="Delete"
        okButtonProps={{ danger: true, loading: deleteLoading }}
      >
        <p>Are you sure you want to delete <strong>{deleteTarget?.displayName || deleteTarget?.samAccountName}</strong>?</p>
        <p>This action cannot be undone.</p>
      </Modal>
    </Space>
  );
}
