import {
  Drawer, Descriptions, Tag, Space, Button, Typography, Divider, Tooltip,
  notification,
} from 'antd';
import {
  LockOutlined, StopOutlined, CheckCircleOutlined, KeyOutlined,
  CopyOutlined, MailOutlined,
} from '@ant-design/icons';

const { Text, Title } = Typography;

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

interface UserDrawerProps {
  user: User | null;
  open: boolean;
  onClose: () => void;
}

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

export default function UserDrawer({ user, open, onClose }: UserDrawerProps) {
  if (!user) return null;

  const statusTag = user.lockedOut
    ? <Tag icon={<LockOutlined />} color="error">Locked Out</Tag>
    : !user.enabled
      ? <Tag icon={<StopOutlined />} color="default">Disabled</Tag>
      : <Tag icon={<CheckCircleOutlined />} color="success">Active</Tag>;

  return (
    <Drawer
      title={
        <Space>
          <span>{user.displayName}</span>
          {statusTag}
        </Space>
      }
      placement="right"
      width={560}
      open={open}
      onClose={onClose}
      extra={
        <Space>
          <Button icon={<KeyOutlined />} onClick={() => notification.info({ message: 'Reset password — not yet implemented' })}>
            Reset Password
          </Button>
          {user.lockedOut && (
            <Button type="primary" onClick={() => notification.info({ message: 'Unlock — not yet implemented' })}>
              Unlock
            </Button>
          )}
        </Space>
      }
    >
      {/* Identity */}
      <Title level={5} style={{ marginBottom: 12 }}>Identity</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="Display Name">{user.displayName}</Descriptions.Item>
        <Descriptions.Item label="First Name">{user.givenName}</Descriptions.Item>
        <Descriptions.Item label="Last Name">{user.sn}</Descriptions.Item>
        <Descriptions.Item label="Username">
          <Space>
            <Text code>{user.samAccountName}</Text>
            <Tooltip title="Copy username">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(user.samAccountName, 'Username')}
              />
            </Tooltip>
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="UPN">
          <Space>
            <Text code>{user.userPrincipalName}</Text>
            <Tooltip title="Copy UPN">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(user.userPrincipalName, 'UPN')}
              />
            </Tooltip>
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="Email">
          <Space>
            <MailOutlined />
            <a href={`mailto:${user.mail}`}>{user.mail}</a>
          </Space>
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Organization */}
      <Title level={5} style={{ marginBottom: 12 }}>Organization</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="Department">{user.department}</Descriptions.Item>
        <Descriptions.Item label="Title">{user.title}</Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Account */}
      <Title level={5} style={{ marginBottom: 12 }}>Account</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="Status">{statusTag}</Descriptions.Item>
        <Descriptions.Item label="Last Logon">
          <Tooltip title={new Date(user.lastLogon).toLocaleString()}>
            {new Date(user.lastLogon).toLocaleString()}
          </Tooltip>
        </Descriptions.Item>
        <Descriptions.Item label="Created">
          {new Date(user.whenCreated).toLocaleDateString()}
        </Descriptions.Item>
        <Descriptions.Item label="DN">
          <Space>
            <Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>{user.dn}</Text>
            <Tooltip title="Copy DN">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(user.dn, 'DN')}
              />
            </Tooltip>
          </Space>
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Group Memberships */}
      <Title level={5} style={{ marginBottom: 12 }}>Group Memberships ({user.memberOf.length})</Title>
      <Space size={[4, 8]} wrap>
        {user.memberOf.map((group) => (
          <Tag key={group} style={{ cursor: 'pointer' }}>{group}</Tag>
        ))}
      </Space>

      <Divider />

      {/* Actions */}
      <Space direction="vertical" style={{ width: '100%' }}>
        {user.enabled ? (
          <Button block danger onClick={() => notification.info({ message: 'Disable — not yet implemented' })}>
            Disable Account
          </Button>
        ) : (
          <Button block type="primary" onClick={() => notification.info({ message: 'Enable — not yet implemented' })}>
            Enable Account
          </Button>
        )}
        <Button block danger type="primary" onClick={() => notification.info({ message: 'Delete — not yet implemented' })}>
          Delete User
        </Button>
      </Space>
    </Drawer>
  );
}
