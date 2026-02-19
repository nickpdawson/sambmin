import { useState } from 'react';
import {
  Card, Form, Input, Select, Button, Table, Tag, Space, Typography, Alert, Row, Col,
  notification,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { SearchOutlined, CopyOutlined } from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface DNSRecord {
  name: string;
  type: string;
  value: string;
  ttl: number;
  priority?: number;
  weight?: number;
  port?: number;
  dynamic: boolean;
}

interface QueryResult {
  server: string;
  zone: string;
  name: string;
  records: DNSRecord[];
  error?: string;
}

const recordTypes = ['ALL', 'A', 'AAAA', 'CNAME', 'MX', 'SRV', 'TXT', 'NS', 'SOA', 'PTR'];
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

const typeColors: Record<string, string> = {
  A: 'blue', AAAA: 'blue', CNAME: 'purple', MX: 'green',
  SRV: 'orange', TXT: 'cyan', NS: 'magenta', SOA: 'red', PTR: 'gold',
};

export default function QueryToolTab() {
  const [form] = Form.useForm();
  const [result, setResult] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(false);

  const handleQuery = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const data = await api.post<QueryResult>('/dns/query', {
        server: values.server || '',
        zone: values.zone,
        name: values.name || '@',
        type: values.type || 'ALL',
      });
      setResult(data);
      if (data.error) {
        notification.warning({ message: 'Query returned error', description: data.error });
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Query failed';
      notification.error({ message: 'DNS query failed', description: msg });
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    notification.success({ message: 'Copied', duration: 2, placement: 'bottomRight' });
  };

  const columns: ColumnsType<DNSRecord> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (name: string) => <Text style={mono}>{name}</Text>,
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (type: string) => <Tag color={typeColors[type] || 'default'}>{type}</Tag>,
    },
    {
      title: 'Value',
      dataIndex: 'value',
      key: 'value',
      ellipsis: true,
      render: (value: string) => (
        <Space>
          <Text style={{ ...mono, fontSize: 12 }}>{value}</Text>
          <CopyOutlined
            style={{ color: 'var(--ant-color-text-tertiary)', cursor: 'pointer', fontSize: 12 }}
            onClick={() => handleCopy(value)}
          />
        </Space>
      ),
    },
    {
      title: 'TTL',
      dataIndex: 'ttl',
      key: 'ttl',
      width: 80,
      align: 'right' as const,
      render: (ttl: number) => <Text style={{ ...mono, fontSize: 12 }}>{ttl}</Text>,
    },
  ];

  // Build CLI preview
  const allValues = Form.useWatch([], form);
  const cliPreview = `samba-tool dns query ${allValues?.server || 'localhost'} ${allValues?.zone || '<zone>'} ${allValues?.name || '@'} ${allValues?.type || 'ALL'}`;

  return (
    <div>
      <Card title="DNS Query Tool" style={{ marginBottom: 16 }}>
        <Form form={form} layout="vertical" initialValues={{ type: 'ALL', name: '@' }}>
          <Row gutter={16}>
            <Col span={6}>
              <Form.Item name="server" label="Server (DC)">
                <Input
                  placeholder="localhost"
                  style={mono}
                />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="zone"
                label="Zone"
                rules={[{ required: true, message: 'Zone is required' }]}
              >
                <Input placeholder="dzsec.net" style={mono} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="name"
                label="Record Name"
              >
                <Input placeholder="@ (zone root)" style={mono} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item name="type" label="Record Type">
                <Select options={recordTypes.map((t) => ({ value: t, label: t }))} />
              </Form.Item>
            </Col>
          </Row>
          <Space>
            <Button
              type="primary"
              icon={<SearchOutlined />}
              loading={loading}
              onClick={handleQuery}
            >
              Query
            </Button>
          </Space>
        </Form>
      </Card>

      {result && (
        <>
          {result.error && (
            <Alert
              type="error"
              message="Query Error"
              description={result.error}
              style={{ marginBottom: 16 }}
            />
          )}

          <Card
            title={`Results — ${result.name} @ ${result.zone}`}
            extra={
              <Text type="secondary" style={{ fontSize: 12 }}>
                Server: <Text style={{ ...mono, fontSize: 12 }}>{result.server}</Text>
                {' | '}
                {result.records.length} record{result.records.length !== 1 ? 's' : ''}
              </Text>
            }
          >
            <Table
              columns={columns}
              dataSource={result.records}
              rowKey={(r) => `${r.name}-${r.type}-${r.value}`}
              pagination={false}
              size="small"
              locale={{ emptyText: result.error ? 'Query returned an error' : 'No records found' }}
            />
          </Card>
        </>
      )}

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent:</Text>
          <Text
            copyable
            style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
          >
            {cliPreview}
          </Text>
        </Space>
      </Card>
    </div>
  );
}
