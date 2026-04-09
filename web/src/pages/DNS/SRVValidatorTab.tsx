import { useState, useEffect, useMemo } from 'react';
import {
  Card, Table, Tag, Spin, Alert, Space, Typography, Row, Col, Statistic, Button, Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ReloadOutlined, CheckCircleOutlined, WarningOutlined, ExclamationCircleOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface SRVValidationEntry {
  record: string;
  dc: string;
  status: string;
  targets: number;
  message?: string;
}

interface SRVValidatorResponse {
  entries: SRVValidationEntry[];
  records: string[];
  dcs: string[];
  summary: {
    total: number;
    passed: number;
    failed: number;
    errors: number;
  };
}

const statusIcon: Record<string, React.ReactNode> = {
  pass: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
  fail: <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />,
  error: <WarningOutlined style={{ color: '#faad14' }} />,
};

export default function SRVValidatorTab() {
  const [data, setData] = useState<SRVValidatorResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = () => {
    setLoading(true);
    setError(null);
    api.get<SRVValidatorResponse>('/dns/srv-validator')
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchData(); }, []);

  // Build matrix: rows = SRV records, columns = DCs
  const matrixData = useMemo(() => {
    if (!data) return [];
    return data.records.map((record) => {
      const row: Record<string, unknown> = { record };
      data.dcs.forEach((dc) => {
        const entry = data.entries.find((e) => e.record === record && e.dc === dc);
        row[dc] = entry;
      });
      return row;
    });
  }, [data]);

  const columns: ColumnsType<Record<string, unknown>> = useMemo(() => {
    if (!data) return [];
    return [
      {
        title: 'SRV Record',
        dataIndex: 'record',
        key: 'record',
        width: 240,
        fixed: 'left' as const,
        render: (record: string) => (
          <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>{record}</Text>
        ),
      },
      ...data.dcs.map((dc) => ({
        title: dc,
        dataIndex: dc,
        key: dc,
        width: 160,
        align: 'center' as const,
        render: (entry: SRVValidationEntry | undefined) => {
          if (!entry) return <Tag>N/A</Tag>;
          return (
            <Tooltip title={entry.message || ''}>
              <Space size={4}>
                {statusIcon[entry.status] || entry.status}
                {entry.status === 'pass' && (
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    {entry.targets} target{entry.targets !== 1 ? 's' : ''}
                  </Text>
                )}
                {entry.status === 'fail' && (
                  <Text type="danger" style={{ fontSize: 11 }}>Missing</Text>
                )}
                {entry.status === 'error' && (
                  <Text type="warning" style={{ fontSize: 11 }}>Error</Text>
                )}
              </Space>
            </Tooltip>
          );
        },
      })),
    ];
  }, [data]);

  if (loading) return <Spin style={{ display: 'block', margin: '48px auto' }} />;
  if (error) return <Alert type="error" message="SRV validation failed" description={error} />;
  if (!data) return null;

  const allPassed = data.summary.failed === 0 && data.summary.errors === 0;

  return (
    <div>
      {!allPassed && (
        <Alert
          type="warning"
          showIcon
          message="SRV Record Issues Detected"
          description={`${data.summary.failed} failed and ${data.summary.errors} errors across ${data.dcs.length} DC${data.dcs.length !== 1 ? 's' : ''}`}
          style={{ marginBottom: 16 }}
        />
      )}

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Passed"
              value={data.summary.passed}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Failed"
              value={data.summary.failed}
              valueStyle={{ color: data.summary.failed > 0 ? '#ff4d4f' : undefined }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Errors"
              value={data.summary.errors}
              valueStyle={{ color: data.summary.errors > 0 ? '#faad14' : undefined }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="DCs Checked" value={data.dcs.length} />
          </Card>
        </Col>
      </Row>

      <Card
        title="SRV Record Validation Matrix"
        extra={<Button icon={<ReloadOutlined />} size="small" onClick={fetchData}>Refresh</Button>}
      >
        <Table
          columns={columns}
          dataSource={matrixData}
          rowKey="record"
          pagination={false}
          size="small"
          scroll={{ x: 240 + data.dcs.length * 160 }}
        />
      </Card>

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space direction="vertical" size={4}>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
          {data.records.slice(0, 3).map((rec) => (
            <Text
              key={rec}
              copyable
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
            >
              samba-tool dns query dc1.example.com example.com {rec} SRV
            </Text>
          ))}
          {data.records.length > 3 && (
            <Text type="secondary" style={{ fontSize: 11 }}>
              ...and {data.records.length - 3} more SRV records
            </Text>
          )}
        </Space>
      </Card>
    </div>
  );
}
