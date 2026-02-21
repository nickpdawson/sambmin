import { useEffect, useState } from 'react';
import { Card, Col, Row, Statistic, Typography, Space, Tag, Alert, Timeline, Skeleton } from 'antd';
import {
  UserOutlined,
  DesktopOutlined,
  TeamOutlined,
  GlobalOutlined,
  LockOutlined,
  PlusOutlined,
  KeyOutlined,
  UnlockOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';
import SelfServiceDashboard from './SelfServiceDashboard';

const { Title, Text } = Typography;

interface DCStatus {
  hostname: string;
  address: string;
  site: string;
  status: string;
  lastReplication: string;
  fsmoRoles: string[] | null;
  isGlobalCatalog: boolean;
}

interface DashboardAlert {
  severity: string;
  message: string;
}

interface Metrics {
  totalUsers: number;
  totalComputers: number;
  totalGroups: number;
  totalDNSZones: number;
  lockedAccounts: number;
  disabledUsers: number;
}

interface Activity {
  timestamp: string;
  actor: string;
  action: string;
  object: string;
  success: boolean;
}

const statusColors: Record<string, string> = {
  healthy: '#16A34A',
  warning: '#D97706',
  error: '#DC2626',
  unreachable: '#94A3B8',
};

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function AdminDashboard() {
  const navigate = useNavigate();
  const [dcs, setDCs] = useState<DCStatus[]>([]);
  const [alerts, setAlerts] = useState<DashboardAlert[]>([]);
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [healthData, metricsData, activityData] = await Promise.all([
          api.get<{ domainControllers: DCStatus[]; alerts: DashboardAlert[] }>('/dashboard/health'),
          api.get<Metrics>('/dashboard/metrics'),
          api.get<{ activities: Activity[] }>('/dashboard/activity'),
        ]);
        setDCs(healthData.domainControllers);
        setAlerts(healthData.alerts);
        setMetrics(metricsData);
        setActivities(activityData.activities);
      } catch {
        // API not available — use empty state
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  const quickActions = [
    { icon: <PlusOutlined />, label: 'Create User', path: '/users' },
    { icon: <KeyOutlined />, label: 'Reset Password', path: '/users' },
    { icon: <GlobalOutlined />, label: 'DNS Record', path: '/dns' },
    { icon: <UnlockOutlined />, label: 'Unlock Account', badge: metrics?.lockedAccounts, path: '/users' },
    { icon: <TeamOutlined />, label: 'Add to Group', path: '/groups' },
    { icon: <DesktopOutlined />, label: 'Find Computer', path: '/computers' },
  ];

  if (loading) {
    return <Skeleton active paragraph={{ rows: 12 }} />;
  }

  return (
    <Space direction="vertical" size={24} style={{ width: '100%' }}>
      {/* DC Health Strip */}
      <Row gutter={[16, 16]}>
        {dcs.map((dc) => (
          <Col key={dc.hostname} xs={24} sm={8}>
            <Card size="small" hoverable>
              <Space>
                <div
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    backgroundColor: statusColors[dc.status],
                    flexShrink: 0,
                  }}
                />
                <div style={{ minWidth: 0 }}>
                  <Text strong style={{ fontFamily: '"JetBrains Mono", monospace', fontSize: 13 }}>
                    {dc.hostname}
                  </Text>
                  <br />
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {dc.site} · Last sync: {timeAgo(dc.lastReplication)}
                  </Text>
                </div>
                <Tag color={dc.status === 'healthy' ? 'success' : dc.status === 'warning' ? 'warning' : 'error'}>
                  {dc.status}
                </Tag>
              </Space>
              {dc.fsmoRoles && dc.fsmoRoles.length > 0 && (
                <div style={{ marginTop: 8 }}>
                  {dc.fsmoRoles.map((role) => (
                    <Tag key={role} style={{ fontSize: 11, marginBottom: 2 }}>{role}</Tag>
                  ))}
                </div>
              )}
            </Card>
          </Col>
        ))}
      </Row>

      {/* Alert banners */}
      {alerts.map((alert, i) => (
        <Alert
          key={i}
          message={alert.message}
          type={alert.severity === 'warning' ? 'warning' : alert.severity === 'error' ? 'error' : 'info'}
          showIcon
          closable
          banner
        />
      ))}

      {/* Quick Actions + Metrics */}
      <Row gutter={24}>
        <Col xs={24} lg={14}>
          <Title level={5} style={{ marginBottom: 16 }}>Quick Actions</Title>
          <Row gutter={[12, 12]}>
            {quickActions.map((action) => (
              <Col key={action.label} xs={12} sm={8}>
                <Card
                  hoverable
                  size="small"
                  style={{ textAlign: 'center', cursor: 'pointer' }}
                  onClick={() => navigate(action.path)}
                >
                  <Space direction="vertical" size={4}>
                    <span style={{ fontSize: 20 }}>{action.icon}</span>
                    <Text style={{ fontSize: 13 }}>{action.label}</Text>
                    {action.badge != null && action.badge > 0 && (
                      <Tag color="error" style={{ margin: 0 }}>{action.badge} locked</Tag>
                    )}
                  </Space>
                </Card>
              </Col>
            ))}
          </Row>
        </Col>

        <Col xs={24} lg={10}>
          <Title level={5} style={{ marginBottom: 16 }}>Domain Metrics</Title>
          {metrics && (
            <Row gutter={[16, 16]}>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="Users" value={metrics.totalUsers} prefix={<UserOutlined />} />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="Computers" value={metrics.totalComputers} prefix={<DesktopOutlined />} />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="Groups" value={metrics.totalGroups} prefix={<TeamOutlined />} />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="DNS Zones" value={metrics.totalDNSZones} prefix={<GlobalOutlined />} />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic
                    title="Locked Out"
                    value={metrics.lockedAccounts}
                    prefix={<LockOutlined />}
                    valueStyle={{ color: metrics.lockedAccounts > 0 ? '#DC2626' : undefined }}
                  />
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small">
                  <Statistic title="Disabled" value={metrics.disabledUsers} prefix={<UserOutlined />} />
                </Card>
              </Col>
            </Row>
          )}
        </Col>
      </Row>

      {/* Recent Activity */}
      {activities.length > 0 && (
        <div>
          <Title level={5} style={{ marginBottom: 16 }}>Recent Activity</Title>
          <Card size="small">
            <Timeline
              items={activities.map((a) => ({
                color: a.success ? 'green' : 'red',
                dot: a.success ? <CheckCircleOutlined /> : <CloseCircleOutlined />,
                children: (
                  <div>
                    <Text strong>{a.action}</Text>
                    <br />
                    <Text code style={{ fontSize: 12 }}>{a.object}</Text>
                    <br />
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {a.actor} · {timeAgo(a.timestamp)}
                    </Text>
                  </div>
                ),
              }))}
            />
          </Card>
        </div>
      )}
    </Space>
  );
}

export default function Dashboard() {
  const { isAdmin } = useAuth();

  if (!isAdmin) {
    return <SelfServiceDashboard />;
  }

  return <AdminDashboard />;
}
