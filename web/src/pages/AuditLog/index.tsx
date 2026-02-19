import { useState, useEffect, useMemo } from 'react';
import {
  Typography, Table, Tag, Card, Space, Button, Row, Col, Statistic, Alert,
  Input, Select,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AuditOutlined, ReloadOutlined, CheckCircleOutlined, ExclamationCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';
import ExportButton from '../../components/ExportButton';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 12 };

interface AuditEntry {
  id: number;
  timestamp: string;
  actor: string;
  action: string;
  objectDN: string;
  objectType: string;
  dc: string;
  success: boolean;
  details: string;
  sourceIP: string;
}

export default function AuditLog() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [resultFilter, setResultFilter] = useState<string>('all');

  const fetchEntries = () => {
    setLoading(true);
    setError(null);
    api.get<{ entries: AuditEntry[]; total: number }>('/audit?limit=500')
      .then((data) => {
        setEntries(data.entries || []);
        setTotal(data.total);
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchEntries(); }, []);

  const filteredEntries = useMemo(() => {
    return entries.filter((e) => {
      if (typeFilter !== 'all' && e.objectType !== typeFilter) return false;
      if (resultFilter === 'success' && !e.success) return false;
      if (resultFilter === 'failed' && e.success) return false;
      if (search) {
        const q = search.toLowerCase();
        return (
          e.actor.toLowerCase().includes(q) ||
          e.action.toLowerCase().includes(q) ||
          e.objectDN.toLowerCase().includes(q) ||
          e.details.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [entries, search, typeFilter, resultFilter]);

  const objectTypes = useMemo(() => {
    const types = new Set(entries.map((e) => e.objectType).filter(Boolean));
    return Array.from(types).sort();
  }, [entries]);

  const formatTime = (ts: string) => {
    const d = new Date(ts);
    return d.toLocaleString();
  };

  const typeColors: Record<string, string> = {
    user: 'blue',
    group: 'purple',
    computer: 'cyan',
    contact: 'green',
    dns: 'orange',
    ou: 'magenta',
    gpo: 'geekblue',
    spn: 'gold',
    delegation: 'lime',
    replication: 'volcano',
    site: 'default',
    fsmo: 'red',
  };

  const columns: ColumnsType<AuditEntry> = [
    {
      title: 'Time',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (ts: string) => <Text style={mono}>{formatTime(ts)}</Text>,
      sorter: (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
      defaultSortOrder: 'descend',
    },
    {
      title: 'Actor',
      dataIndex: 'actor',
      key: 'actor',
      width: 120,
      render: (actor: string) => <Text strong>{actor}</Text>,
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      width: 160,
    },
    {
      title: 'Type',
      dataIndex: 'objectType',
      key: 'objectType',
      width: 100,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>{type}</Tag>
      ),
    },
    {
      title: 'Object',
      dataIndex: 'objectDN',
      key: 'objectDN',
      ellipsis: true,
      render: (dn: string) => <Text style={mono}>{dn}</Text>,
    },
    {
      title: 'Result',
      dataIndex: 'success',
      key: 'success',
      width: 90,
      align: 'center' as const,
      render: (success: boolean) =>
        success
          ? <Tag color="green" icon={<CheckCircleOutlined />}>OK</Tag>
          : <Tag color="red" icon={<ExclamationCircleOutlined />}>Failed</Tag>,
    },
    {
      title: 'Details',
      dataIndex: 'details',
      key: 'details',
      ellipsis: true,
      render: (d: string) => d ? <Text style={{ fontSize: 12 }}>{d}</Text> : null,
    },
  ];

  const successCount = filteredEntries.filter((e) => e.success).length;
  const failCount = filteredEntries.filter((e) => !e.success).length;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <AuditOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Audit Log</Title>
        </Space>
        <Space>
          <ExportButton
            data={filteredEntries as unknown as Record<string, unknown>[]}
            filename="sambmin-audit"
            columns={[
              { key: 'timestamp', title: 'Timestamp' },
              { key: 'actor', title: 'Actor' },
              { key: 'action', title: 'Action' },
              { key: 'objectType', title: 'Type' },
              { key: 'objectDN', title: 'Object' },
              { key: 'success', title: 'Success' },
              { key: 'details', title: 'Details' },
              { key: 'dc', title: 'DC' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchEntries}>Refresh</Button>
        </Space>
      </div>

      {error && (
        <Alert type="error" message="Failed to load audit log" description={error} style={{ marginBottom: 16 }} />
      )}

      {/* Filters */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input
            placeholder="Search actions, actors, objects..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            allowClear
            style={{ width: 280 }}
          />
          <Select
            value={typeFilter}
            onChange={setTypeFilter}
            style={{ width: 140 }}
            options={[
              { value: 'all', label: 'All types' },
              ...objectTypes.map((t) => ({ value: t, label: t })),
            ]}
          />
          <Select
            value={resultFilter}
            onChange={setResultFilter}
            style={{ width: 140 }}
            options={[
              { value: 'all', label: 'All results' },
              { value: 'success', label: 'Success only' },
              { value: 'failed', label: 'Failed only' },
            ]}
          />
          {(search || typeFilter !== 'all' || resultFilter !== 'all') && (
            <Button
              size="small"
              onClick={() => {
                setSearch('');
                setTypeFilter('all');
                setResultFilter('all');
              }}
            >
              Clear filters
            </Button>
          )}
        </Space>
      </Card>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic title="Total Entries" value={total} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Successful"
              value={successCount}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="Failed"
              value={failCount}
              valueStyle={failCount > 0 ? { color: '#ff4d4f' } : undefined}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="Shown" value={filteredEntries.length} suffix={`/ ${total}`} />
          </Card>
        </Col>
      </Row>

      <Table
        columns={columns}
        dataSource={filteredEntries}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 50, showSizeChanger: true, pageSizeOptions: ['25', '50', '100', '200'] }}
        size="middle"
      />

      {entries.length === 0 && !loading && !error && (
        <Alert
          type="info"
          message="No audit entries yet"
          description="Administrative actions will appear here as they are performed."
          style={{ marginTop: 16 }}
        />
      )}
    </div>
  );
}
