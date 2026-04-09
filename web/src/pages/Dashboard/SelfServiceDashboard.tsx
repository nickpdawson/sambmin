import { useEffect, useState, useCallback } from 'react';
import {
  Card, Col, Row, Typography, Space, Tag, Descriptions, Button,
  Form, Input, notification, Skeleton, Alert,
} from 'antd';
import {
  UserOutlined, KeyOutlined, MailOutlined,
  PhoneOutlined, TeamOutlined, CheckCircleOutlined,
  ClockCircleOutlined, EditOutlined, SaveOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';

const { Title, Text } = Typography;

const MONO: React.CSSProperties = { fontFamily: "'JetBrains Mono', monospace", fontSize: 12 };

interface UserProfile {
  dn: string;
  samAccountName: string;
  displayName: string;
  givenName: string;
  sn: string;
  mail: string;
  userPrincipalName: string;
  department: string;
  title: string;
  company: string;
  office: string;
  phone: string;
  mobile: string;
  enabled: boolean;
  lockedOut: boolean;
  passwordExpired: boolean;
  accountExpires: string;
  pwdLastSet: string;
  lastLogon: string;
  whenCreated: string;
  memberOf: string[];
}

const cnFromDN = (dn: string) => dn.split(',')[0]?.replace(/^CN=/i, '') || dn;

function formatDate(iso: string): string {
  if (!iso) return 'Never';
  const d = new Date(iso);
  if (d.getFullYear() < 1971) return 'Never';
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export default function SelfServiceDashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [editForm] = Form.useForm();
  const [editLoading, setEditLoading] = useState(false);
  const [pwForm] = Form.useForm();
  const [pwLoading, setPwLoading] = useState(false);

  const loadProfile = useCallback(async () => {
    try {
      const data = await api.get<UserProfile>('/self');
      setProfile(data);
    } catch {
      // not available
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const handleEditSave = useCallback(async () => {
    try {
      const values = await editForm.validateFields();
      setEditLoading(true);
      await api.put('/self', values);
      notification.success({ message: 'Profile updated' });
      setEditing(false);
      loadProfile();
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        notification.error({ message: 'Update failed', description: err.message });
      }
    } finally {
      setEditLoading(false);
    }
  }, [editForm, loadProfile]);

  const handlePasswordChange = useCallback(async () => {
    try {
      const values = await pwForm.validateFields();
      if (values.newPassword !== values.confirmPassword) {
        notification.error({ message: 'Passwords do not match' });
        return;
      }
      setPwLoading(true);
      await api.post('/self/password', {
        currentPassword: values.currentPassword,
        newPassword: values.newPassword,
      });
      notification.success({ message: 'Password changed successfully' });
      pwForm.resetFields();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Password change failed';
      notification.error({ message: 'Password change failed', description: msg });
    } finally {
      setPwLoading(false);
    }
  }, [pwForm]);

  if (loading) {
    return <Skeleton active paragraph={{ rows: 10 }} />;
  }

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      {/* Welcome */}
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          Welcome, {profile?.givenName || user?.username}
        </Title>
        <Text type="secondary">Manage your account and browse the directory</Text>
      </div>

      {/* Account alerts */}
      {profile?.lockedOut && (
        <Alert type="error" message="Your account is locked out. Contact an administrator." showIcon />
      )}
      {profile?.passwordExpired && (
        <Alert type="warning" message="Your password has expired. Please change it below." showIcon />
      )}

      <Row gutter={[24, 24]}>
        {/* Profile Card */}
        <Col xs={24} lg={14}>
          <Card
            title={
              <Space>
                <UserOutlined />
                <span>My Profile</span>
              </Space>
            }
            extra={
              !editing ? (
                <Button
                  icon={<EditOutlined />}
                  onClick={() => {
                    editForm.setFieldsValue({
                      phone: profile?.phone || '',
                      mobile: profile?.mobile || '',
                      department: profile?.department || '',
                      title: profile?.title || '',
                      office: profile?.office || '',
                    });
                    setEditing(true);
                  }}
                >
                  Edit
                </Button>
              ) : (
                <Space>
                  <Button icon={<CloseOutlined />} onClick={() => setEditing(false)}>Cancel</Button>
                  <Button type="primary" icon={<SaveOutlined />} loading={editLoading} onClick={handleEditSave}>
                    Save
                  </Button>
                </Space>
              )
            }
          >
            {!editing ? (
              <Descriptions column={1} size="small" bordered>
                <Descriptions.Item label="Display Name">{profile?.displayName}</Descriptions.Item>
                <Descriptions.Item label="Username">
                  <Text code style={MONO}>{profile?.samAccountName}</Text>
                </Descriptions.Item>
                <Descriptions.Item label="Email">
                  <Space>
                    <MailOutlined />
                    <span>{profile?.mail || <Text type="secondary">Not set</Text>}</span>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="Department">{profile?.department || <Text type="secondary">Not set</Text>}</Descriptions.Item>
                <Descriptions.Item label="Title">{profile?.title || <Text type="secondary">Not set</Text>}</Descriptions.Item>
                <Descriptions.Item label="Office">{profile?.office || <Text type="secondary">Not set</Text>}</Descriptions.Item>
                <Descriptions.Item label="Phone">
                  <Space>
                    <PhoneOutlined />
                    <span>{profile?.phone || <Text type="secondary">Not set</Text>}</span>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="Mobile">{profile?.mobile || <Text type="secondary">Not set</Text>}</Descriptions.Item>
              </Descriptions>
            ) : (
              <Form form={editForm} layout="vertical">
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item name="phone" label="Phone">
                      <Input prefix={<PhoneOutlined />} placeholder="Phone number" />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item name="mobile" label="Mobile">
                      <Input placeholder="Mobile number" />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item name="department" label="Department">
                      <Input placeholder="Department" />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item name="title" label="Title">
                      <Input placeholder="Job title" />
                    </Form.Item>
                  </Col>
                  <Col span={24}>
                    <Form.Item name="office" label="Office">
                      <Input placeholder="Office location" />
                    </Form.Item>
                  </Col>
                </Row>
              </Form>
            )}
          </Card>
        </Col>

        {/* Account Info + Quick Actions */}
        <Col xs={24} lg={10}>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {/* Account Status */}
            <Card
              title={
                <Space>
                  <CheckCircleOutlined />
                  <span>Account Status</span>
                </Space>
              }
              size="small"
            >
              <Descriptions column={1} size="small">
                <Descriptions.Item label="Status">
                  {profile?.enabled ? (
                    <Tag color="success" icon={<CheckCircleOutlined />}>Active</Tag>
                  ) : (
                    <Tag color="error">Disabled</Tag>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="Password Set">
                  <Space>
                    <ClockCircleOutlined />
                    <span>{formatDate(profile?.pwdLastSet || '')}</span>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="Last Logon">{formatDate(profile?.lastLogon || '')}</Descriptions.Item>
                <Descriptions.Item label="Account Created">{formatDate(profile?.whenCreated || '')}</Descriptions.Item>
              </Descriptions>
            </Card>

            {/* Group Memberships */}
            <Card
              title={
                <Space>
                  <TeamOutlined />
                  <span>My Groups ({profile?.memberOf?.length || 0})</span>
                </Space>
              }
              size="small"
            >
              {(profile?.memberOf || []).length > 0 ? (
                <Space size={[4, 8]} wrap>
                  {profile!.memberOf.map((g) => (
                    <Tag key={g}>{cnFromDN(g)}</Tag>
                  ))}
                </Space>
              ) : (
                <Text type="secondary">No group memberships</Text>
              )}
            </Card>

            {/* Quick Links */}
            <Card title="Quick Links" size="small">
              <Space direction="vertical" style={{ width: '100%' }}>
                <Button block icon={<UserOutlined />} onClick={() => navigate('/users')}>
                  Browse Users
                </Button>
                <Button block icon={<TeamOutlined />} onClick={() => navigate('/groups')}>
                  Browse Groups
                </Button>
              </Space>
            </Card>
          </Space>
        </Col>
      </Row>

      {/* Change Password */}
      <Card
        title={
          <Space>
            <KeyOutlined />
            <span>Change Password</span>
          </Space>
        }
      >
        <Form form={pwForm} layout="vertical" style={{ maxWidth: 400 }}>
          <Form.Item
            name="currentPassword"
            label="Current Password"
            rules={[{ required: true, message: 'Current password is required' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label="New Password"
            rules={[
              { required: true, message: 'New password is required' },
              { min: 12, message: 'Must be at least 12 characters' },
            ]}
          >
            <Input.Password placeholder="Minimum 12 characters" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="Confirm New Password"
            rules={[{ required: true, message: 'Please confirm your new password' }]}
          >
            <Input.Password />
          </Form.Item>
          <Button type="primary" icon={<KeyOutlined />} loading={pwLoading} onClick={handlePasswordChange}>
            Change Password
          </Button>
        </Form>
      </Card>
    </Space>
  );
}
