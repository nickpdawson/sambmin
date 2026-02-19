import { useState, useEffect, useMemo } from 'react';
import {
  Card, Table, Tag, Spin, Alert, Space, Typography, Row, Col, Statistic, Button, Select,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ReloadOutlined, CheckCircleOutlined, ExclamationCircleOutlined, WarningOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface DCResult {
  dc: string;
  soaSerial: number;
  records: number;
  status: string;
  error?: string;
}

interface ConsistencyResponse {
  zone: string;
  consistent: boolean;
  dcs: DCResult[];
}

interface DNSZone {
  name: string;
}

export default function ConsistencyTab() {
  const [data, setData] = useState<ConsistencyResponse | null>(null);
  const [zones, setZones] = useState<DNSZone[]>([]);
  const [selectedZone, setSelectedZone] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load zone list for the dropdown
  useEffect(() => {
    api.get<{ zones: DNSZone[] }>('/dns/zones')
      .then((d) => setZones(d.zones))
      .catch(() => {});
  }, []);

  const fetchData = (zone?: string) => {
    setLoading(true);
    setError(null);
    const params = zone ? `?zone=${encodeURIComponent(zone)}` : '';
    api.get<ConsistencyResponse>(`/dns/consistency${params}`)
      .then((d) => {
        setData(d);
        setSelectedZone(d.zone);
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(); }, []);

  // Find the max serial to highlight divergence
  const maxSerial = useMemo(() => {
    if (!data) return 0;
    return Math.max(...data.dcs.filter((d) => d.status === 'ok').map((d) => d.soaSerial));
  }, [data]);

  const columns: ColumnsType<DCResult> = [
    {
      title: 'DC',
      dataIndex: 'dc',
      key: 'dc',
      width: 200,
      render: (dc: string) => (
        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{dc}</Text>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        if (status === 'ok') return <Tag color="green" icon={<CheckCircleOutlined />}>OK</Tag>;
        return <Tag color="red" icon={<ExclamationCircleOutlined />}>Error</Tag>;
      },
    },
    {
      title: 'SOA Serial',
      dataIndex: 'soaSerial',
      key: 'soaSerial',
      width: 150,
      render: (serial: number, record: DCResult) => {
        if (record.status === 'error') return <Text type="secondary">---</Text>;
        const isLatest = serial === maxSerial;
        return (
          <Space>
            <Text
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              type={isLatest ? undefined : 'warning'}
            >
              {serial}
            </Text>
            {!isLatest && <Tag color="orange">Behind</Tag>}
          </Space>
        );
      },
    },
    {
      title: 'Records',
      dataIndex: 'records',
      key: 'records',
      width: 100,
      align: 'right' as const,
      render: (count: number, record: DCResult) => {
        if (record.status === 'error' || count < 0) return <Text type="secondary">---</Text>;
        return count;
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

  if (loading) return <Spin style={{ display: 'block', margin: '48px auto' }} />;
  if (error) return <Alert type="error" message="Consistency check failed" description={error} />;
  if (!data) return null;

  const errorCount = data.dcs.filter((d) => d.status === 'error').length;

  return (
    <div>
      {!data.consistent && (
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message="DNS Inconsistency Detected"
          description="SOA serial numbers differ across domain controllers. This may indicate replication lag."
          style={{ marginBottom: 16 }}
        />
      )}

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Consistency"
              value={data.consistent ? 'In Sync' : 'Diverged'}
              valueStyle={{
                color: data.consistent ? '#52c41a' : '#ff4d4f',
                fontSize: 16,
              }}
              prefix={data.consistent ? <CheckCircleOutlined /> : <WarningOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="DCs Checked" value={data.dcs.length} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Unreachable"
              value={errorCount}
              valueStyle={errorCount > 0 ? { color: '#ff4d4f' } : undefined}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Latest Serial"
              value={maxSerial}
              valueStyle={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 16 }}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={`DNS Consistency — ${data.zone}`}
        extra={
          <Space>
            <Select
              value={selectedZone}
              onChange={(zone) => fetchData(zone)}
              style={{ width: 220 }}
              size="small"
              options={zones.map((z) => ({ value: z.name, label: z.name }))}
              placeholder="Select zone"
            />
            <Button icon={<ReloadOutlined />} size="small" onClick={() => fetchData(selectedZone)}>
              Refresh
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data.dcs}
          rowKey="dc"
          pagination={false}
          size="middle"
        />
      </Card>

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space direction="vertical" size={4}>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
          <Text
            copyable
            style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
          >
            samba-tool dns query dc1.dzsec.net {data.zone} @ SOA
          </Text>
          <Text
            copyable
            style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
          >
            dig @dc1.dzsec.net {data.zone} SOA +short
          </Text>
        </Space>
      </Card>
    </div>
  );
}
