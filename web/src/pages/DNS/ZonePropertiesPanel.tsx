import { useState, useEffect } from 'react';
import {
  Card, Descriptions, Tag, Space, Typography, Button, Switch, InputNumber, notification, Spin,
  Collapse,
} from 'antd';
import { SettingOutlined, ReloadOutlined } from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface DNSZoneInfo {
  name: string;
  type: string;
  backend: string;
  dynamicUpdate: string;
  agingEnabled: boolean;
  noRefreshInterval: number;
  refreshInterval: number;
  scavengeServers: string;
  records: number;
  soaSerial: number;
  status: string;
}

interface ZonePropertiesPanelProps {
  zoneName: string;
}

export default function ZonePropertiesPanel({ zoneName }: ZonePropertiesPanelProps) {
  const [info, setInfo] = useState<DNSZoneInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editAging, setEditAging] = useState(false);
  const [agingForm, setAgingForm] = useState({
    aging: false,
    noRefreshInterval: 168,
    refreshInterval: 168,
  });

  const fetchInfo = () => {
    setLoading(true);
    api.get<DNSZoneInfo>(`/dns/zones/${encodeURIComponent(zoneName)}/info`)
      .then((data) => {
        setInfo(data);
        setAgingForm({
          aging: data.agingEnabled,
          noRefreshInterval: data.noRefreshInterval,
          refreshInterval: data.refreshInterval,
        });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchInfo(); }, [zoneName]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.put(`/dns/zones/${encodeURIComponent(zoneName)}/options`, {
        aging: agingForm.aging,
        noRefreshInterval: agingForm.noRefreshInterval,
        refreshInterval: agingForm.refreshInterval,
      });
      notification.success({
        message: 'Zone options updated',
        description: `Aging/scavenging settings saved for ${zoneName}`,
      });
      setEditAging(false);
      fetchInfo();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Save failed';
      notification.error({ message: 'Update failed', description: msg });
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Spin size="small" />;
  if (!info) return null;

  return (
    <Collapse
      ghost
      items={[
        {
          key: 'properties',
          label: (
            <Space>
              <SettingOutlined />
              <Text strong style={{ fontSize: 13 }}>Zone Properties</Text>
              {info.agingEnabled && <Tag color="blue" style={{ fontSize: 11 }}>Aging Enabled</Tag>}
              {info.status !== 'healthy' && <Tag color="orange">{info.status}</Tag>}
            </Space>
          ),
          children: (
            <div>
              <Descriptions column={3} size="small" bordered>
                <Descriptions.Item label="Type">
                  <Tag color={info.type === 'forward' ? 'blue' : 'purple'}>{info.type}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Backend">
                  <Tag color={info.backend === 'samba' ? 'green' : 'orange'}>{info.backend}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Dynamic Updates">
                  <Tag color={info.dynamicUpdate === 'secure' ? 'green' : info.dynamicUpdate === 'none' ? 'default' : 'orange'}>
                    {info.dynamicUpdate}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="SOA Serial">
                  <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                    {info.soaSerial}
                  </Text>
                </Descriptions.Item>
                <Descriptions.Item label="Records">{info.records}</Descriptions.Item>
                <Descriptions.Item label="Status">
                  <Tag color={info.status === 'healthy' ? 'green' : info.status === 'stale' ? 'red' : 'orange'}>
                    {info.status}
                  </Tag>
                </Descriptions.Item>
              </Descriptions>

              {/* Aging/Scavenging Section */}
              <Card
                size="small"
                title="Aging / Scavenging"
                style={{ marginTop: 12 }}
                extra={
                  editAging ? (
                    <Space>
                      <Button size="small" onClick={() => setEditAging(false)}>Cancel</Button>
                      <Button size="small" type="primary" loading={saving} onClick={handleSave}>
                        Save
                      </Button>
                    </Space>
                  ) : (
                    <Space>
                      <Button icon={<ReloadOutlined />} size="small" onClick={fetchInfo} />
                      <Button size="small" onClick={() => setEditAging(true)}>Edit</Button>
                    </Space>
                  )
                }
              >
                {editAging ? (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Space>
                      <Text>Aging:</Text>
                      <Switch
                        checked={agingForm.aging}
                        onChange={(v) => setAgingForm({ ...agingForm, aging: v })}
                      />
                    </Space>
                    <Space>
                      <Text>No-refresh interval (hours):</Text>
                      <InputNumber
                        min={0}
                        value={agingForm.noRefreshInterval}
                        onChange={(v) => setAgingForm({ ...agingForm, noRefreshInterval: v ?? 0 })}
                      />
                    </Space>
                    <Space>
                      <Text>Refresh interval (hours):</Text>
                      <InputNumber
                        min={0}
                        value={agingForm.refreshInterval}
                        onChange={(v) => setAgingForm({ ...agingForm, refreshInterval: v ?? 0 })}
                      />
                    </Space>
                  </Space>
                ) : (
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label="Aging">
                      <Tag color={info.agingEnabled ? 'blue' : 'default'}>
                        {info.agingEnabled ? 'Enabled' : 'Disabled'}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="Scavenge Servers">
                      <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                        {info.scavengeServers || 'None'}
                      </Text>
                    </Descriptions.Item>
                    <Descriptions.Item label="No-Refresh Interval">
                      {info.noRefreshInterval} hours ({Math.round(info.noRefreshInterval / 24)} days)
                    </Descriptions.Item>
                    <Descriptions.Item label="Refresh Interval">
                      {info.refreshInterval} hours ({Math.round(info.refreshInterval / 24)} days)
                    </Descriptions.Item>
                  </Descriptions>
                )}
              </Card>

              {/* CLI equivalent */}
              <Card size="small" style={{ marginTop: 8 }}>
                <Space>
                  <Text type="secondary" style={{ fontSize: 12 }}>CLI:</Text>
                  <Text
                    copyable
                    style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
                  >
                    samba-tool dns zoneinfo localhost {zoneName}
                  </Text>
                </Space>
              </Card>
            </div>
          ),
        },
      ]}
      style={{ marginBottom: 12 }}
    />
  );
}
