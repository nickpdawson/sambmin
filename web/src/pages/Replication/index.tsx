import { useState, useEffect } from 'react';
import {
  Typography, Tabs, Card, Table, Tag, Space, Button, Row, Col, Statistic,
  Alert, Tooltip, Modal, Select, notification,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SyncOutlined, CheckCircleOutlined, WarningOutlined, ExclamationCircleOutlined,
  ReloadOutlined, ThunderboltOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

interface ReplicationLink {
  sourceDC: string;
  destDC: string;
  namingContext: string;
  lastSync: string;
  status: string;
  pendingChanges: number;
}

interface TopoResponse {
  links: ReplicationLink[];
  dcs: string[];
}

interface DCReplStatus {
  dc: string;
  address: string;
  site: string;
  reachable: boolean;
  inboundOk: number;
  inboundFailed: number;
  outboundOk: number;
  outboundFailed: number;
  lastSuccess: string;
  error?: string;
}

interface StatusResponse {
  dcs: DCReplStatus[];
  summary: {
    total: number;
    healthy: number;
    degraded: number;
    failed: number;
  };
}

export default function Replication() {
  const [topo, setTopo] = useState<TopoResponse | null>(null);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [activeTab, setActiveTab] = useState('status');
  const [loading, setLoading] = useState(true);
  const [syncModalOpen, setSyncModalOpen] = useState(false);
  const [syncSource, setSyncSource] = useState('');
  const [syncDest, setSyncDest] = useState('');
  const [syncing, setSyncing] = useState(false);

  const fetchStatus = () => {
    setLoading(true);
    api.get<StatusResponse>('/replication/status')
      .then(setStatus)
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  const fetchTopo = () => {
    api.get<TopoResponse>('/replication/topology')
      .then(setTopo)
      .catch(() => {});
  };

  useEffect(() => {
    fetchStatus();
    fetchTopo();
  }, []);

  const handleSync = async () => {
    if (!syncSource || !syncDest) return;
    setSyncing(true);
    try {
      await api.post('/replication/sync', {
        sourceDC: syncSource,
        destDC: syncDest,
      });
      notification.success({
        message: 'Sync triggered',
        description: `Replication from ${syncSource} to ${syncDest} initiated`,
      });
      setSyncModalOpen(false);
      fetchStatus();
      fetchTopo();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Sync failed';
      notification.error({ message: 'Sync failed', description: msg });
    } finally {
      setSyncing(false);
    }
  };

  const statusColumns: ColumnsType<DCReplStatus> = [
    {
      title: 'DC',
      dataIndex: 'dc',
      key: 'dc',
      render: (dc: string) => <Text style={mono}>{dc}</Text>,
    },
    {
      title: 'Site',
      dataIndex: 'site',
      key: 'site',
      width: 160,
    },
    {
      title: 'Status',
      key: 'status',
      width: 120,
      render: (_: unknown, record: DCReplStatus) => {
        if (!record.reachable) return <Tag color="red" icon={<ExclamationCircleOutlined />}>Unreachable</Tag>;
        if (record.inboundFailed > 0 || record.outboundFailed > 0)
          return <Tag color="orange" icon={<WarningOutlined />}>Degraded</Tag>;
        return <Tag color="green" icon={<CheckCircleOutlined />}>Healthy</Tag>;
      },
    },
    {
      title: 'Inbound',
      key: 'inbound',
      width: 100,
      align: 'center' as const,
      render: (_: unknown, record: DCReplStatus) => {
        if (!record.reachable) return '---';
        return (
          <Space size={4}>
            <Text type={record.inboundOk > 0 ? undefined : 'secondary'}>{record.inboundOk}</Text>
            {record.inboundFailed > 0 && (
              <Text type="danger">({record.inboundFailed} fail)</Text>
            )}
          </Space>
        );
      },
    },
    {
      title: 'Outbound',
      key: 'outbound',
      width: 100,
      align: 'center' as const,
      render: (_: unknown, record: DCReplStatus) => {
        if (!record.reachable) return '---';
        return (
          <Space size={4}>
            <Text>{record.outboundOk}</Text>
            {record.outboundFailed > 0 && (
              <Text type="danger">({record.outboundFailed} fail)</Text>
            )}
          </Space>
        );
      },
    },
    {
      title: 'Error',
      dataIndex: 'error',
      key: 'error',
      ellipsis: true,
      render: (err: string | undefined) =>
        err ? <Text type="danger" style={{ fontSize: 12 }}>{err}</Text> : null,
    },
  ];

  const linkColumns: ColumnsType<ReplicationLink> = [
    {
      title: 'Source',
      dataIndex: 'sourceDC',
      key: 'source',
      render: (dc: string) => <Text style={mono}>{dc}</Text>,
    },
    {
      title: '',
      key: 'arrow',
      width: 40,
      align: 'center' as const,
      render: () => <SyncOutlined />,
    },
    {
      title: 'Destination',
      dataIndex: 'destDC',
      key: 'dest',
      render: (dc: string) => <Text style={mono}>{dc}</Text>,
    },
    {
      title: 'Naming Context',
      dataIndex: 'namingContext',
      key: 'nc',
      ellipsis: true,
      render: (nc: string) => (
        <Tooltip title={nc}>
          <Text style={{ ...mono, fontSize: 12 }}>{nc}</Text>
        </Tooltip>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => {
        const color = s === 'current' ? 'green' : s === 'behind' ? 'orange' : 'red';
        return <Tag color={color}>{s}</Tag>;
      },
    },
  ];

  const dcOptions = (topo?.dcs || []).map((dc) => ({ value: dc, label: dc }));

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <SyncOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Replication</Title>
        </Space>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => { fetchStatus(); fetchTopo(); }}>
            Refresh
          </Button>
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={() => setSyncModalOpen(true)}
          >
            Force Sync
          </Button>
        </Space>
      </div>

      {status && status.summary.failed > 0 && (
        <Alert
          type="error"
          showIcon
          message="Replication Issues"
          description={`${status.summary.failed} DC${status.summary.failed > 1 ? 's' : ''} unreachable or failing replication`}
          style={{ marginBottom: 16 }}
        />
      )}

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'status',
            label: 'DC Status',
            children: (
              <div>
                {status && (
                  <Row gutter={16} style={{ marginBottom: 16 }}>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic title="Total DCs" value={status.summary.total} />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Healthy"
                          value={status.summary.healthy}
                          valueStyle={{ color: '#52c41a' }}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Degraded"
                          value={status.summary.degraded}
                          valueStyle={status.summary.degraded > 0 ? { color: '#faad14' } : undefined}
                          prefix={<WarningOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Failed"
                          value={status.summary.failed}
                          valueStyle={status.summary.failed > 0 ? { color: '#ff4d4f' } : undefined}
                          prefix={<ExclamationCircleOutlined />}
                        />
                      </Card>
                    </Col>
                  </Row>
                )}

                <Table
                  columns={statusColumns}
                  dataSource={status?.dcs || []}
                  rowKey="dc"
                  loading={loading}
                  pagination={false}
                  size="middle"
                />

                <Card size="small" style={{ marginTop: 16 }}>
                  <Space>
                    <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent:</Text>
                    <Text copyable style={mono}>
                      samba-tool drs showrepl
                    </Text>
                  </Space>
                </Card>
              </div>
            ),
          },
          {
            key: 'topology',
            label: `Topology${topo ? ` (${topo.links.length} links)` : ''}`,
            children: (
              <div>
                <Card title="Replication Links">
                  <Table
                    columns={linkColumns}
                    dataSource={topo?.links || []}
                    rowKey={(r) => `${r.sourceDC}-${r.destDC}-${r.namingContext}`}
                    pagination={false}
                    size="middle"
                  />
                </Card>

                {topo && topo.dcs.length > 0 && (
                  <Card size="small" title="Domain Controllers" style={{ marginTop: 16 }}>
                    <Space wrap>
                      {topo.dcs.map((dc) => (
                        <Tag key={dc} color="blue">
                          <Text style={mono}>{dc}</Text>
                        </Tag>
                      ))}
                    </Space>
                  </Card>
                )}
              </div>
            ),
          },
        ]}
      />

      {/* Force Sync Modal */}
      <Modal
        title="Force Replication Sync"
        open={syncModalOpen}
        onCancel={() => setSyncModalOpen(false)}
        onOk={handleSync}
        okText="Force Sync"
        confirmLoading={syncing}
        okButtonProps={{ danger: true }}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Text>
            Force replication between two domain controllers. This triggers an immediate sync.
          </Text>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Source DC:</Text>
            <Select
              value={syncSource || undefined}
              onChange={setSyncSource}
              options={dcOptions}
              placeholder="Select source DC"
              style={{ width: '100%' }}
            />
          </div>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Destination DC:</Text>
            <Select
              value={syncDest || undefined}
              onChange={setSyncDest}
              options={dcOptions}
              placeholder="Select destination DC"
              style={{ width: '100%' }}
            />
          </div>
          <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
            <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
            <Text copyable style={{ ...mono, fontSize: 12 }}>
              samba-tool drs replicate {syncDest || '<dest>'} {syncSource || '<source>'} DC=example,DC=com
            </Text>
          </Card>
        </Space>
      </Modal>
    </div>
  );
}
