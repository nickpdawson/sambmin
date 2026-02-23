import { useState, useEffect, useCallback } from 'react';
import {
  Drawer, Descriptions, Tag, Space, Button, Typography, Tooltip, Tabs,
  notification, Modal, Select, Input,
} from 'antd';
import {
  LockOutlined, StopOutlined, CheckCircleOutlined, KeyOutlined,
  CopyOutlined, EditOutlined, SaveOutlined,
  CloseOutlined, PlusOutlined, DeleteOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Text, Title } = Typography;

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

interface Group {
  dn: string;
  name: string;
  samAccountName: string;
  description: string;
}

interface UserDrawerProps {
  user: User | null;
  open: boolean;
  onClose: () => void;
  onRefresh?: () => void;
}

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

function cnFromDN(dn: string): string {
  const parts = dn.split(',');
  if (parts.length === 0) return dn;
  const cn = parts[0];
  if (cn.toUpperCase().startsWith('CN=')) return cn.substring(3);
  return cn;
}

function formatTimestamp(iso: string): string {
  if (!iso) return 'Never';
  const d = new Date(iso);
  if (d.getFullYear() < 1971) return 'Never';
  return d.toLocaleString();
}

function timeAgo(iso: string): string {
  if (!iso) return 'Never';
  const d = new Date(iso);
  if (d.getFullYear() < 1971) return 'Never';
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

// Editable field content — renders inside a Descriptions.Item provided by parent
function EditableFieldContent({ value, fieldName, onSave }: {
  value: string;
  fieldName: string;
  onSave: (field: string, value: string) => Promise<void>;
}) {
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(value);
  const [saving, setSaving] = useState(false);

  useEffect(() => { setEditValue(value); }, [value]);

  const handleSave = async () => {
    if (editValue === value) { setEditing(false); return; }
    setSaving(true);
    try {
      await onSave(fieldName, editValue);
      setEditing(false);
    } finally {
      setSaving(false);
    }
  };

  if (editing) {
    return (
      <Space>
        <Input
          size="small"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onPressEnter={handleSave}
          style={{ width: 200 }}
          autoFocus
        />
        <Button type="text" size="small" icon={<SaveOutlined />} loading={saving} onClick={handleSave} />
        <Button type="text" size="small" icon={<CloseOutlined />} onClick={() => { setEditValue(value); setEditing(false); }} />
      </Space>
    );
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
      <span>{value || <Text type="secondary">—</Text>}</span>
      <Button
        size="small"
        icon={<EditOutlined />}
        onClick={() => setEditing(true)}
      >
        Edit
      </Button>
    </div>
  );
}

export default function UserDrawer({ user, open, onClose, onRefresh }: UserDrawerProps) {
  const [activeTab, setActiveTab] = useState('identity');
  const [allGroups, setAllGroups] = useState<Group[]>([]);
  const [addGroupOpen, setAddGroupOpen] = useState(false);
  const [selectedGroupDn, setSelectedGroupDn] = useState<string | null>(null);
  const [addingGroup, setAddingGroup] = useState(false);

  // Load available groups when Groups tab is opened
  useEffect(() => {
    if (activeTab === 'groups' && allGroups.length === 0) {
      api.get<{ groups: Group[] }>('/groups')
        .then(data => setAllGroups(data.groups))
        .catch(() => {});
    }
  }, [activeTab, allGroups.length]);

  const handleFieldSave = useCallback(async (fieldName: string, value: string) => {
    if (!user) return;
    const dn = encodeURIComponent(user.dn);
    try {
      await api.put(`/users/${dn}`, { [fieldName]: value });
      notification.success({ message: `${fieldName} updated`, duration: 2 });
      onRefresh?.();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Update failed';
      Modal.error({ title: 'Update failed', content: msg });
    }
  }, [user, onRefresh]);

  const handleAddGroup = useCallback(async () => {
    if (!user || !selectedGroupDn) return;
    setAddingGroup(true);
    try {
      const groupDn = encodeURIComponent(selectedGroupDn);
      await api.post(`/groups/${groupDn}/members`, { memberDn: user.dn });
      notification.success({ message: `Added to ${cnFromDN(selectedGroupDn)}` });
      setAddGroupOpen(false);
      setSelectedGroupDn(null);
      onRefresh?.();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to add to group';
      Modal.error({ title: 'Add to group failed', content: msg });
    } finally {
      setAddingGroup(false);
    }
  }, [user, selectedGroupDn, onRefresh]);

  const handleRemoveGroup = useCallback(async (groupDn: string) => {
    if (!user) return;
    Modal.confirm({
      title: 'Remove from group',
      icon: <ExclamationCircleOutlined />,
      content: `Remove ${user.displayName || user.samAccountName} from ${cnFromDN(groupDn)}?`,
      okText: 'Remove',
      okButtonProps: { danger: true },
      onOk: async () => {
        const gdn = encodeURIComponent(groupDn);
        const mdn = encodeURIComponent(user.dn);
        await api.delete(`/groups/${gdn}/members/${mdn}`);
        notification.success({ message: `Removed from ${cnFromDN(groupDn)}` });
        onRefresh?.();
      },
    });
  }, [user, onRefresh]);

  const handleUserAction = useCallback(async (action: string) => {
    if (!user) return;
    const dn = encodeURIComponent(user.dn);
    const name = user.displayName || user.samAccountName;

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
      onRefresh?.();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Operation failed';
      Modal.error({ title: `Failed to ${action}`, content: msg });
    }
  }, [user, onRefresh]);

  if (!user) return null;

  const statusTag = user.lockedOut
    ? <Tag icon={<LockOutlined />} color="error">Locked Out</Tag>
    : !user.enabled
      ? <Tag icon={<StopOutlined />} color="default">Disabled</Tag>
      : <Tag icon={<CheckCircleOutlined />} color="success">Active</Tag>;

  // Groups not already a member of (for the add dropdown)
  const availableGroups = allGroups.filter(g => !user.memberOf.includes(g.dn));

  const tabItems = [
    {
      key: 'identity',
      label: 'Identity',
      children: (
        <>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Display Name"><EditableFieldContent value={user.displayName} fieldName="displayName" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="First Name"><EditableFieldContent value={user.givenName} fieldName="givenName" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Last Name"><EditableFieldContent value={user.sn} fieldName="surname" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Username">
              <Space>
                <Text code>{user.samAccountName}</Text>
                <Tooltip title="Copy"><Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(user.samAccountName, 'Username')} /></Tooltip>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="UPN">
              <Space>
                <Text code>{user.userPrincipalName}</Text>
                <Tooltip title="Copy"><Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(user.userPrincipalName, 'UPN')} /></Tooltip>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="Email"><EditableFieldContent value={user.mail} fieldName="mail" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Description"><EditableFieldContent value={user.description} fieldName="description" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Phone"><EditableFieldContent value={user.phone} fieldName="phone" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Mobile"><EditableFieldContent value={user.mobile} fieldName="mobile" onSave={handleFieldSave} /></Descriptions.Item>
          </Descriptions>
        </>
      ),
    },
    {
      key: 'organization',
      label: 'Organization',
      children: (
        <>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Title"><EditableFieldContent value={user.title} fieldName="title" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Department"><EditableFieldContent value={user.department} fieldName="department" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Company"><EditableFieldContent value={user.company} fieldName="company" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Manager">
              {user.manager ? (
                <Space>
                  <span>{cnFromDN(user.manager)}</span>
                  <Tooltip title="Copy DN"><Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(user.manager, 'Manager DN')} /></Tooltip>
                </Space>
              ) : <Text type="secondary">—</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="Office"><EditableFieldContent value={user.office} fieldName="office" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Street"><EditableFieldContent value={user.streetAddress} fieldName="streetAddress" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="City"><EditableFieldContent value={user.city} fieldName="city" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="State"><EditableFieldContent value={user.state} fieldName="state" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Postal Code"><EditableFieldContent value={user.postalCode} fieldName="postalCode" onSave={handleFieldSave} /></Descriptions.Item>
            <Descriptions.Item label="Country"><EditableFieldContent value={user.country} fieldName="country" onSave={handleFieldSave} /></Descriptions.Item>
          </Descriptions>
        </>
      ),
    },
    {
      key: 'account',
      label: 'Account',
      children: (
        <>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Status">{statusTag}</Descriptions.Item>
            <Descriptions.Item label="Last Logon">
              <Tooltip title={formatTimestamp(user.lastLogon)}>
                {timeAgo(user.lastLogon)}
              </Tooltip>
            </Descriptions.Item>
            <Descriptions.Item label="Password Last Set">
              <Tooltip title={formatTimestamp(user.pwdLastSet)}>
                {timeAgo(user.pwdLastSet)}
              </Tooltip>
            </Descriptions.Item>
            <Descriptions.Item label="Password Expired">
              {user.passwordExpired
                ? <Tag color="warning">Must change at next login</Tag>
                : <Text type="secondary">No</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="Bad Password Count">
              {user.badPwdCount > 0
                ? <Tag color="warning">{user.badPwdCount}</Tag>
                : <Text type="secondary">0</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="Account Expires">
              {formatTimestamp(user.accountExpires) === 'Never'
                ? <Text type="secondary">Never</Text>
                : formatTimestamp(user.accountExpires)}
            </Descriptions.Item>
            <Descriptions.Item label="Created">
              {formatTimestamp(user.whenCreated)}
            </Descriptions.Item>
            <Descriptions.Item label="Modified">
              <Tooltip title={formatTimestamp(user.whenChanged)}>
                {timeAgo(user.whenChanged)}
              </Tooltip>
            </Descriptions.Item>
            <Descriptions.Item label="DN">
              <Space>
                <Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>{user.dn}</Text>
                <Tooltip title="Copy DN"><Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(user.dn, 'DN')} /></Tooltip>
              </Space>
            </Descriptions.Item>
          </Descriptions>

          <div style={{ marginTop: 16 }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              {user.lockedOut && (
                <Button block type="primary" onClick={() => handleUserAction('unlock')}>
                  Unlock Account
                </Button>
              )}
              {user.enabled ? (
                <Button block danger onClick={() => handleUserAction('disable')}>
                  Disable Account
                </Button>
              ) : (
                <Button block type="primary" onClick={() => handleUserAction('enable')}>
                  Enable Account
                </Button>
              )}
            </Space>
          </div>
        </>
      ),
    },
    {
      key: 'groups',
      label: `Groups (${user.memberOf.length})`,
      children: (
        <>
          <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Title level={5} style={{ margin: 0 }}>Group Memberships</Title>
            <Button size="small" icon={<PlusOutlined />} onClick={() => setAddGroupOpen(true)}>
              Add to Group
            </Button>
          </div>

          {user.memberOf.length === 0 ? (
            <Text type="secondary">No group memberships</Text>
          ) : (
            <Space direction="vertical" style={{ width: '100%' }} size={4}>
              {user.memberOf.map((groupDn) => (
                <div key={groupDn} style={{
                  display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                  padding: '6px 8px', borderRadius: 6,
                  border: '1px solid var(--ant-color-border-secondary, #f0f0f0)',
                }}>
                  <Space>
                    <Tag>{cnFromDN(groupDn)}</Tag>
                    <Text type="secondary" style={{ fontSize: 11, fontFamily: 'JetBrains Mono, monospace' }}>
                      {groupDn}
                    </Text>
                  </Space>
                  <Tooltip title="Remove from group">
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => handleRemoveGroup(groupDn)}
                    />
                  </Tooltip>
                </div>
              ))}
            </Space>
          )}

          {/* Add to group modal */}
          <Modal
            title="Add to Group"
            open={addGroupOpen}
            onCancel={() => { setAddGroupOpen(false); setSelectedGroupDn(null); }}
            onOk={handleAddGroup}
            okText="Add"
            confirmLoading={addingGroup}
            okButtonProps={{ disabled: !selectedGroupDn }}
          >
            <Select
              showSearch
              placeholder="Search and select a group..."
              style={{ width: '100%' }}
              value={selectedGroupDn}
              onChange={(v) => setSelectedGroupDn(v)}
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              options={availableGroups.map(g => ({
                label: g.name || g.samAccountName,
                value: g.dn,
              }))}
            />
          </Modal>
        </>
      ),
    },
  ];

  return (
    <Drawer
      title={
        <Space>
          <span>{user.displayName || user.samAccountName}</span>
          {statusTag}
        </Space>
      }
      placement="right"
      width={600}
      open={open}
      onClose={onClose}
      extra={
        <Space>
          <Button icon={<KeyOutlined />} onClick={() => {
            // Trigger reset password from parent — emit a custom event
            onClose();
            // The parent index.tsx handles reset password modal
          }}>
            Reset Password
          </Button>
          {user.lockedOut && (
            <Button type="primary" onClick={() => handleUserAction('unlock')}>
              Unlock
            </Button>
          )}
        </Space>
      }
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={tabItems}
        size="small"
      />
    </Drawer>
  );
}
