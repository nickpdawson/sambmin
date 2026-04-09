import { useState, useEffect } from 'react';
import {
  Modal, Button, Space, Table, Tag, Typography, Input, Select, Popconfirm,
  notification, Alert, Form,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, TeamOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { api } from '../../api/client';

const { Text } = Typography;

interface RoleEntry {
  role: string;
  groups: string[];
  permissions: string[];
}

interface RbacModalProps {
  open: boolean;
  roles: RoleEntry[];
  onClose: () => void;
  onSave: (roles: RoleEntry[]) => void;
}

// Available permissions for the select dropdown
const AVAILABLE_PERMISSIONS = [
  'users:read', 'users:write', 'users:delete',
  'groups:read', 'groups:write', 'groups:delete',
  'dns:read', 'dns:write', 'dns:delete',
  'gpo:read', 'gpo:write', 'gpo:delete',
  'settings:read', 'settings:write',
  'audit:read',
  'computers:read', 'computers:write', 'computers:delete',
  '*',
];

// Common AD groups for the select dropdown
const AVAILABLE_GROUPS = [
  'Domain Admins',
  'Domain Users',
  'Enterprise Admins',
  'Schema Admins',
  'DNS Admins',
  'Server Operators',
  'Account Operators',
  'Backup Operators',
  'IT-Ops',
  'Help Desk',
  'Security Team',
];

export default function RbacModal({ open, roles, onClose, onSave }: RbacModalProps) {
  const [editableRoles, setEditableRoles] = useState<RoleEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [addingRole, setAddingRole] = useState(false);
  const [newRoleForm] = Form.useForm();

  useEffect(() => {
    if (open) {
      setEditableRoles(roles.map((r) => ({ ...r, groups: [...r.groups], permissions: [...r.permissions] })));
      setAddingRole(false);
      newRoleForm.resetFields();
    }
  }, [open, roles, newRoleForm]);

  const handleGroupsChange = (roleName: string, groups: string[]) => {
    setEditableRoles((prev) =>
      prev.map((r) => (r.role === roleName ? { ...r, groups } : r))
    );
  };

  const handlePermissionsChange = (roleName: string, permissions: string[]) => {
    setEditableRoles((prev) =>
      prev.map((r) => (r.role === roleName ? { ...r, permissions } : r))
    );
  };

  const handleDeleteRole = (roleName: string) => {
    setEditableRoles((prev) => prev.filter((r) => r.role !== roleName));
  };

  const handleAddRole = async () => {
    try {
      const values = await newRoleForm.validateFields();
      const exists = editableRoles.some(
        (r) => r.role.toLowerCase() === values.roleName.toLowerCase()
      );
      if (exists) {
        notification.error({ message: 'A role with that name already exists.' });
        return;
      }
      setEditableRoles((prev) => [
        ...prev,
        {
          role: values.roleName,
          groups: values.groups || [],
          permissions: values.permissions || [],
        },
      ]);
      newRoleForm.resetFields();
      setAddingRole(false);
    } catch {
      // Validation failed
    }
  };

  const handleSave = async () => {
    setLoading(true);
    try {
      await api.put('/settings/rbac', { roles: editableRoles });
      onSave(editableRoles);
      notification.success({ message: 'RBAC configuration saved. Changes take effect on next login.' });
    } catch (err: any) {
      notification.error({ message: err?.message || 'Failed to save RBAC configuration.' });
    } finally {
      setLoading(false);
    }
  };

  const columns: ColumnsType<RoleEntry> = [
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      width: 140,
      render: (role: string) => <Text strong>{role}</Text>,
    },
    {
      title: 'AD Groups',
      dataIndex: 'groups',
      key: 'groups',
      render: (groups: string[], record) => (
        <Select
          mode="multiple"
          value={groups}
          onChange={(val) => handleGroupsChange(record.role, val)}
          style={{ width: '100%' }}
          placeholder="Add AD groups..."
          options={AVAILABLE_GROUPS.map((g) => ({ value: g, label: g }))}
          tagRender={({ label, closable, onClose: onTagClose }) => (
            <Tag color="blue" closable={closable} onClose={onTagClose} style={{ marginRight: 3 }}>
              {label}
            </Tag>
          )}
        />
      ),
    },
    {
      title: 'Permissions',
      dataIndex: 'permissions',
      key: 'permissions',
      render: (permissions: string[], record) => (
        <Select
          mode="multiple"
          value={permissions}
          onChange={(val) => handlePermissionsChange(record.role, val)}
          style={{ width: '100%' }}
          placeholder="Add permissions..."
          options={AVAILABLE_PERMISSIONS.map((p) => ({ value: p, label: p }))}
          tagRender={({ label, closable, onClose: onTagClose }) => (
            <Tag
              closable={closable}
              onClose={onTagClose}
              style={{
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: 11,
                marginRight: 3,
              }}
            >
              {label}
            </Tag>
          )}
        />
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      render: (_, record) => (
        <Popconfirm
          title={`Delete the "${record.role}" role?`}
          description="This action cannot be undone."
          onConfirm={() => handleDeleteRole(record.role)}
          okText="Delete"
          okButtonProps={{ danger: true }}
        >
          <Button type="text" size="small" icon={<DeleteOutlined />} danger />
        </Popconfirm>
      ),
    },
  ];

  return (
    <Modal
      title={
        <Space>
          <TeamOutlined />
          Edit Role-Based Access Control
        </Space>
      }
      open={open}
      onCancel={onClose}
      width={880}
      okText="Save Changes"
      confirmLoading={loading}
      onOk={handleSave}
    >
      <Alert
        message="Changes to RBAC will take effect on next login"
        description="Users who are currently logged in will retain their existing permissions until their session expires or they log in again."
        type="warning"
        showIcon
        style={{ marginBottom: 16, marginTop: 8 }}
      />

      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          type="dashed"
          icon={<PlusOutlined />}
          onClick={() => setAddingRole(true)}
        >
          Add Role
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={editableRoles}
        rowKey="role"
        pagination={false}
        size="small"
      />

      {addingRole && (
        <div style={{
          marginTop: 12,
          padding: 16,
          background: 'var(--ant-color-bg-layout)',
          borderRadius: 8,
        }}>
          <Text strong style={{ display: 'block', marginBottom: 12 }}>Add New Role</Text>
          <Form form={newRoleForm} layout="vertical">
            <Form.Item
              name="roleName"
              label="Role Name"
              rules={[{ required: true, message: 'Role name is required' }]}
            >
              <Input placeholder="e.g., Auditor" />
            </Form.Item>

            <Space style={{ width: '100%', display: 'flex' }} size={12}>
              <Form.Item name="groups" label="AD Groups" style={{ flex: 1 }}>
                <Select
                  mode="multiple"
                  placeholder="Select AD groups..."
                  options={AVAILABLE_GROUPS.map((g) => ({ value: g, label: g }))}
                />
              </Form.Item>

              <Form.Item name="permissions" label="Permissions" style={{ flex: 1 }}>
                <Select
                  mode="multiple"
                  placeholder="Select permissions..."
                  options={AVAILABLE_PERMISSIONS.map((p) => ({ value: p, label: p }))}
                />
              </Form.Item>
            </Space>

            <Space>
              <Button type="primary" size="small" onClick={handleAddRole}>Add Role</Button>
              <Button size="small" onClick={() => { newRoleForm.resetFields(); setAddingRole(false); }}>Cancel</Button>
            </Space>
          </Form>
        </div>
      )}
    </Modal>
  );
}
