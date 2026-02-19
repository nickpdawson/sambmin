import { useState, useEffect } from 'react';
import {
  Typography, Table, Tag, Card, Space, Button, Row, Col, Statistic, Modal, Input,
  notification, Alert,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ClusterOutlined, PlusOutlined, ReloadOutlined, EnvironmentOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

interface Site {
  name: string;
  subnets: string[];
  dcs: string[];
}

export default function Sites() {
  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedSite, setSelectedSite] = useState<string | null>(null);
  const [subnets, setSubnets] = useState<string[]>([]);
  const [subnetsLoading, setSubnetsLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [newSiteName, setNewSiteName] = useState('');
  const [creating, setCreating] = useState(false);

  const fetchSites = () => {
    setLoading(true);
    setError(null);
    api.get<{ sites: Site[] }>('/sites')
      .then((data) => setSites(data.sites || []))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchSites(); }, []);

  const fetchSubnets = (siteName: string) => {
    setSelectedSite(siteName);
    setSubnetsLoading(true);
    api.get<{ subnets: string[] }>(`/sites/${encodeURIComponent(siteName)}/subnets`)
      .then((data) => setSubnets(data.subnets || []))
      .catch(() => setSubnets([]))
      .finally(() => setSubnetsLoading(false));
  };

  const handleCreate = async () => {
    if (!newSiteName.trim()) return;
    setCreating(true);
    try {
      await api.post('/sites', { name: newSiteName.trim() });
      notification.success({
        message: 'Site created',
        description: `Site "${newSiteName}" created successfully`,
      });
      setCreateOpen(false);
      setNewSiteName('');
      fetchSites();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Create failed';
      notification.error({ message: 'Create failed', description: msg });
    } finally {
      setCreating(false);
    }
  };

  const columns: ColumnsType<Site> = [
    {
      title: 'Site',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <Space>
          <EnvironmentOutlined />
          <a onClick={() => fetchSubnets(name)} style={mono}>{name}</a>
        </Space>
      ),
    },
    {
      title: 'DCs',
      dataIndex: 'dcs',
      key: 'dcs',
      render: (dcs: string[] | null) => (
        <Space wrap>
          {(dcs || []).map((dc) => (
            <Tag key={dc} color="blue">
              <Text style={{ ...mono, fontSize: 12 }}>{dc}</Text>
            </Tag>
          ))}
          {(!dcs || dcs.length === 0) && <Text type="secondary">None assigned</Text>}
        </Space>
      ),
    },
    {
      title: 'Subnets',
      dataIndex: 'subnets',
      key: 'subnets',
      render: (sub: string[] | null) => (
        <Space wrap>
          {(sub || []).map((s) => (
            <Tag key={s}>
              <Text style={{ ...mono, fontSize: 12 }}>{s}</Text>
            </Tag>
          ))}
          {(!sub || sub.length === 0) && <Text type="secondary">---</Text>}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <ClusterOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Sites & Services</Title>
        </Space>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchSites}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            New Site
          </Button>
        </Space>
      </div>

      {error && (
        <Alert type="error" message="Failed to load sites" description={error} style={{ marginBottom: 16 }} />
      )}

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card size="small">
            <Statistic title="Total Sites" value={sites.length} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="Sites with DCs"
              value={sites.filter((s) => s.dcs && s.dcs.length > 0).length}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="Total DCs"
              value={sites.reduce((sum, s) => sum + (s.dcs?.length || 0), 0)}
            />
          </Card>
        </Col>
      </Row>

      <Table
        columns={columns}
        dataSource={sites}
        rowKey="name"
        loading={loading}
        pagination={false}
        size="middle"
        onRow={(record) => ({
          onClick: () => fetchSubnets(record.name),
          style: { cursor: 'pointer' },
        })}
      />

      {/* Subnet detail panel */}
      {selectedSite && (
        <Card
          title={`Subnets — ${selectedSite}`}
          size="small"
          style={{ marginTop: 16 }}
          extra={<Button size="small" onClick={() => setSelectedSite(null)}>Close</Button>}
          loading={subnetsLoading}
        >
          {subnets.length > 0 ? (
            <Space wrap>
              {subnets.map((s) => (
                <Tag key={s} color="blue">
                  <Text style={{ ...mono, fontSize: 12 }}>{s}</Text>
                </Tag>
              ))}
            </Space>
          ) : (
            <Text type="secondary">No subnets configured for this site</Text>
          )}
        </Card>
      )}

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space direction="vertical" size={4}>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool sites list
          </Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool sites subnet list
          </Text>
        </Space>
      </Card>

      {/* Create Site Modal */}
      <Modal
        title="Create Site"
        open={createOpen}
        onCancel={() => { setCreateOpen(false); setNewSiteName(''); }}
        onOk={handleCreate}
        okText="Create"
        confirmLoading={creating}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Site Name:</Text>
            <Input
              value={newSiteName}
              onChange={(e) => setNewSiteName(e.target.value)}
              placeholder="e.g. Bozeman"
              style={mono}
            />
          </div>
          <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
            <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
            <Text copyable style={{ ...mono, fontSize: 12 }}>
              samba-tool sites create {newSiteName || '<name>'}
            </Text>
          </Card>
        </Space>
      </Modal>
    </div>
  );
}
