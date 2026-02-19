import { useState, useEffect } from 'react';
import {
  Card, Descriptions, Tag, Spin, Alert, Space, Typography, Row, Col, Statistic, Button,
} from 'antd';
import {
  ReloadOutlined, CheckCircleOutlined, WarningOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface DNSServerInfo {
  server: string;
  forwarders: string[];
  rootHints: boolean;
  allowUpdate: string;
  zones: number;
  version: string;
}

interface Limitation {
  id: string;
  title: string;
  description: string;
  severity: string;
}

export default function ServerInfoTab() {
  const [info, setInfo] = useState<DNSServerInfo | null>(null);
  const [limitations, setLimitations] = useState<Limitation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = () => {
    setLoading(true);
    setError(null);
    Promise.all([
      api.get<DNSServerInfo>('/dns/serverinfo'),
      api.get<{ limitations: Limitation[] }>('/dns/limitations'),
    ])
      .then(([serverInfo, limData]) => {
        setInfo(serverInfo);
        setLimitations(limData.limitations);
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(); }, []);

  if (loading) return <Spin style={{ display: 'block', margin: '48px auto' }} />;
  if (error) return <Alert type="error" message="Failed to load DNS server info" description={error} />;
  if (!info) return null;

  return (
    <div>
      {/* Limitations banner */}
      {limitations.filter((l) => l.severity === 'warning').length > 0 && (
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message="Samba DNS Limitations"
          description={
            <ul style={{ margin: '4px 0 0', paddingLeft: 20 }}>
              {limitations.filter((l) => l.severity === 'warning').map((l) => (
                <li key={l.id}><Text strong>{l.title}:</Text> {l.description}</li>
              ))}
            </ul>
          }
          style={{ marginBottom: 16 }}
          closable
        />
      )}

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic title="DNS Zones" value={info.zones} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Forwarders"
              value={info.forwarders.length}
              valueStyle={info.forwarders.length === 0 ? { color: '#faad14' } : undefined}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Root Hints"
              value={info.rootHints ? 'Enabled' : 'Disabled'}
              prefix={info.rootHints ? <CheckCircleOutlined /> : <WarningOutlined />}
              valueStyle={{ color: info.rootHints ? '#52c41a' : '#faad14', fontSize: 16 }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Dynamic Updates"
              value={info.allowUpdate || 'Unknown'}
              valueStyle={{ fontSize: 16 }}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title="Server Configuration"
        extra={<Button icon={<ReloadOutlined />} size="small" onClick={fetchData}>Refresh</Button>}
      >
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="Server">
            <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>
              {info.server}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="Version">
            <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>
              {info.version || 'Unknown'}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="Dynamic Updates">
            <Tag color={info.allowUpdate === 'secure' ? 'green' : 'orange'}>
              {info.allowUpdate}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Root Hints">
            <Tag color={info.rootHints ? 'green' : 'default'}>
              {info.rootHints ? 'Enabled' : 'Disabled'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Forwarders" span={2}>
            {info.forwarders.length > 0 ? (
              <Space wrap>
                {info.forwarders.map((fw) => (
                  <Tag key={fw} color="blue">
                    <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                      {fw}
                    </Text>
                  </Tag>
                ))}
              </Space>
            ) : (
              <Text type="secondary">No forwarders configured</Text>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="Total Zones">
            {info.zones}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Info-level limitations */}
      {limitations.filter((l) => l.severity === 'info').length > 0 && (
        <Card title="Additional Notes" size="small" style={{ marginTop: 16 }}>
          {limitations.filter((l) => l.severity === 'info').map((l) => (
            <Alert
              key={l.id}
              type="info"
              message={l.title}
              description={l.description}
              showIcon
              style={{ marginBottom: 8 }}
            />
          ))}
        </Card>
      )}

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent:</Text>
          <Text
            copyable
            style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
          >
            samba-tool dns serverinfo localhost
          </Text>
        </Space>
      </Card>
    </div>
  );
}
