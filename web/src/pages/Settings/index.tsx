import { useState, useEffect, useCallback } from 'react';
import {
  Typography, Card, Descriptions, Tag, Badge, Table, Space, Button,
  Switch, Statistic, Row, Col, Divider, Tooltip, Skeleton,
} from 'antd';
import {
  SettingOutlined, CloudServerOutlined, SafetyCertificateOutlined,
  KeyOutlined, TeamOutlined, AppstoreOutlined, EditOutlined,
  CheckCircleOutlined, WarningOutlined, CopyOutlined, ReloadOutlined,
  LinkOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { api } from '../../api/client';
import ConnectionModal from './ConnectionModal';
import TlsModal from './TlsModal';
import AuthModal from './AuthModal';
import RbacModal from './RbacModal';

const { Title, Text } = Typography;

interface DomainController {
  hostname: string;
  address: string;
  port: number;
  site: string;
  primary: boolean;
  status: string;
}

interface SettingsData {
  connection: {
    domainControllers: DomainController[];
    baseDN: string;
    realm: string;
    protocol: string;
  };
  tls: {
    provider: string;
    domain: string;
    certificate: string;
    key: string;
    expiry: string;
    autoRenew: boolean;
  };
  auth: {
    kerberos: {
      enabled: boolean;
      implementation: string;
      keytab: string;
      spn: string;
    };
    ldapBind: {
      enabled: boolean;
    };
    sessionTimeout: number;
  };
  rbac: {
    roles: Array<{
      role: string;
      groups: string[];
      permissions: string[];
    }>;
  };
  application: {
    version: string;
    scriptsPath: string;
    databaseHost: string;
    databaseName: string;
    auditRetentionDays: number;
  };
}

const dcStatusBadge: Record<string, 'success' | 'warning' | 'error'> = {
  connected: 'success',
  degraded: 'warning',
  disconnected: 'error',
};

export default function Settings() {
  const [settings, setSettings] = useState<SettingsData | null>(null);
  const [loading, setLoading] = useState(true);

  // Modal visibility state
  const [connectionOpen, setConnectionOpen] = useState(false);
  const [tlsOpen, setTlsOpen] = useState(false);
  const [authOpen, setAuthOpen] = useState(false);
  const [rbacOpen, setRbacOpen] = useState(false);

  const loadSettings = useCallback(() => {
    setLoading(true);
    api.get<SettingsData>('/settings')
      .then(setSettings)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  if (loading) {
    return (
      <div>
        <Title level={4}><SettingOutlined /> Settings</Title>
        <Skeleton active paragraph={{ rows: 12 }} />
      </div>
    );
  }

  if (!settings) return null;

  const daysUntilExpiry = Math.ceil(
    (new Date(settings.tls.expiry).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
  );

  const dcColumns: ColumnsType<DomainController> = [
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => <Badge status={dcStatusBadge[status] || 'default'} text={status} />,
    },
    {
      title: 'Hostname',
      dataIndex: 'hostname',
      key: 'hostname',
      render: (hostname: string, record: DomainController) => (
        <Space>
          <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{hostname}</Text>
          {record.primary && <Tag color="blue">Primary</Tag>}
          <Tooltip title="Copy">
            <CopyOutlined
              style={{ color: 'var(--ant-color-text-tertiary)', cursor: 'pointer', fontSize: 12 }}
              onClick={() => navigator.clipboard.writeText(hostname)}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: 'Address',
      dataIndex: 'address',
      key: 'address',
      render: (addr: string) => (
        <Text copyable={{ text: addr }} style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>
          {addr}
        </Text>
      ),
    },
    {
      title: 'Port',
      dataIndex: 'port',
      key: 'port',
      width: 80,
      render: (port: number) => (
        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{port}</Text>
      ),
    },
    {
      title: 'Site',
      dataIndex: 'site',
      key: 'site',
      render: (site: string) => <Tag>{site}</Tag>,
    },
  ];

  const roleColumns: ColumnsType<SettingsData['rbac']['roles'][0]> = [
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      width: 150,
      render: (role: string) => <Text strong>{role}</Text>,
    },
    {
      title: 'AD Groups',
      dataIndex: 'groups',
      key: 'groups',
      render: (groups: string[]) => (
        <Space size={4} wrap>
          {groups.map((g) => <Tag key={g} color="blue">{g}</Tag>)}
        </Space>
      ),
    },
    {
      title: 'Permissions',
      dataIndex: 'permissions',
      key: 'permissions',
      render: (perms: string[]) => (
        <Space size={4} wrap>
          {perms.map((p) => (
            <Tag key={p} style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 11 }}>
              {p}
            </Tag>
          ))}
        </Space>
      ),
    },
  ];

  // Save handlers — update local state optimistically
  const handleConnectionSave = (connection: SettingsData['connection']) => {
    setSettings((prev) => prev ? { ...prev, connection } : prev);
    setConnectionOpen(false);
  };

  const handleTlsSave = (tls: SettingsData['tls']) => {
    setSettings((prev) => prev ? { ...prev, tls } : prev);
    // Don't close — TLS modal manages its own close for the auto-renew toggle
  };

  const handleAuthSave = (auth: SettingsData['auth']) => {
    setSettings((prev) => prev ? { ...prev, auth } : prev);
    setAuthOpen(false);
  };

  const handleRbacSave = (roles: SettingsData['rbac']['roles']) => {
    setSettings((prev) => prev ? { ...prev, rbac: { roles } } : prev);
    setRbacOpen(false);
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Space align="center">
          <SettingOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Settings</Title>
        </Space>
        <Button icon={<ReloadOutlined />} onClick={loadSettings}>Refresh</Button>
      </div>

      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {/* Connection */}
        <Card
          title={<Space><CloudServerOutlined /> Connection</Space>}
          extra={
            <Button type="text" icon={<EditOutlined />} size="small" onClick={() => setConnectionOpen(true)}>
              Edit
            </Button>
          }
        >
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={8}>
              <Statistic title="Realm" value={settings.connection.realm}
                valueStyle={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 16 }} />
            </Col>
            <Col span={8}>
              <Statistic title="Base DN" value={settings.connection.baseDN}
                valueStyle={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 16 }} />
            </Col>
            <Col span={8}>
              <Statistic title="Protocol" value={settings.connection.protocol.toUpperCase()}
                valueStyle={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 16 }} />
            </Col>
          </Row>

          <Divider orientation="left" plain style={{ fontSize: 12 }}>Domain Controllers</Divider>

          <Table
            columns={dcColumns}
            dataSource={settings.connection.domainControllers}
            rowKey="hostname"
            pagination={false}
            size="small"
          />
        </Card>

        {/* TLS */}
        <Card
          title={<Space><SafetyCertificateOutlined /> TLS Certificate</Space>}
          extra={
            <Button type="text" icon={<EditOutlined />} size="small" onClick={() => setTlsOpen(true)}>
              Manage
            </Button>
          }
        >
          <Descriptions column={2} size="small">
            <Descriptions.Item label="Provider">
              <Tag color="green">{settings.tls.provider === 'letsencrypt' ? "Let's Encrypt" : settings.tls.provider}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Domain">
              <Space>
                <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{settings.tls.domain}</Text>
                <LinkOutlined style={{ color: 'var(--ant-color-text-tertiary)', fontSize: 12 }} />
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="Certificate">
              <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>{settings.tls.certificate}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="Key">
              <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>{settings.tls.key}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="Expiry">
              <Space>
                {daysUntilExpiry > 30 ? (
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                ) : (
                  <WarningOutlined style={{ color: '#faad14' }} />
                )}
                <Text>{new Date(settings.tls.expiry).toLocaleDateString()}</Text>
                <Text type="secondary">({daysUntilExpiry} days)</Text>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="Auto-Renew">
              <Switch checked={settings.tls.autoRenew} size="small" disabled />
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* Authentication */}
        <Card
          title={<Space><KeyOutlined /> Authentication</Space>}
          extra={
            <Button type="text" icon={<EditOutlined />} size="small" onClick={() => setAuthOpen(true)}>
              Edit
            </Button>
          }
        >
          <Row gutter={24}>
            <Col span={12}>
              <Card size="small" title="Kerberos / SPNEGO" type="inner">
                <Descriptions column={1} size="small">
                  <Descriptions.Item label="Status">
                    <Badge status={settings.auth.kerberos.enabled ? 'success' : 'default'}
                      text={settings.auth.kerberos.enabled ? 'Enabled' : 'Disabled'} />
                  </Descriptions.Item>
                  <Descriptions.Item label="Implementation">
                    <Tag color="blue">{settings.auth.kerberos.implementation}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="SPN">
                    <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                      {settings.auth.kerberos.spn}
                    </Text>
                  </Descriptions.Item>
                  <Descriptions.Item label="Keytab">
                    <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                      {settings.auth.kerberos.keytab}
                    </Text>
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            </Col>
            <Col span={12}>
              <Card size="small" title="LDAP Bind" type="inner">
                <Descriptions column={1} size="small">
                  <Descriptions.Item label="Status">
                    <Badge status={settings.auth.ldapBind.enabled ? 'success' : 'default'}
                      text={settings.auth.ldapBind.enabled ? 'Enabled' : 'Disabled'} />
                  </Descriptions.Item>
                  <Descriptions.Item label="Session Timeout">
                    <Space>
                      <Text strong>{settings.auth.sessionTimeout}</Text>
                      <Text type="secondary">hours</Text>
                    </Space>
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            </Col>
          </Row>
        </Card>

        {/* RBAC */}
        <Card
          title={<Space><TeamOutlined /> Role-Based Access Control</Space>}
          extra={
            <Button type="text" icon={<EditOutlined />} size="small" onClick={() => setRbacOpen(true)}>
              Edit Roles
            </Button>
          }
        >
          <Table
            columns={roleColumns}
            dataSource={settings.rbac.roles}
            rowKey="role"
            pagination={false}
            size="small"
          />
        </Card>

        {/* Application */}
        <Card
          title={<Space><AppstoreOutlined /> Application</Space>}
        >
          <Descriptions column={2} size="small">
            <Descriptions.Item label="Version">
              <Tag>{settings.application.version}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Audit Retention">
              <Space>
                <Text strong>{settings.application.auditRetentionDays}</Text>
                <Text type="secondary">days</Text>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="Scripts Path">
              <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                {settings.application.scriptsPath}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="Database">
              <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                {settings.application.databaseHost} / {settings.application.databaseName}
              </Text>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      </Space>

      {/* Modals */}
      <ConnectionModal
        open={connectionOpen}
        data={settings.connection}
        onClose={() => setConnectionOpen(false)}
        onSave={handleConnectionSave}
      />

      <TlsModal
        open={tlsOpen}
        data={settings.tls}
        onClose={() => setTlsOpen(false)}
        onSave={handleTlsSave}
      />

      <AuthModal
        open={authOpen}
        data={settings.auth}
        onClose={() => setAuthOpen(false)}
        onSave={handleAuthSave}
      />

      <RbacModal
        open={rbacOpen}
        roles={settings.rbac.roles}
        onClose={() => setRbacOpen(false)}
        onSave={handleRbacSave}
      />
    </div>
  );
}
