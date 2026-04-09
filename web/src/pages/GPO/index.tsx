import { useState, useEffect } from 'react';
import {
  Typography, Table, Tag, Card, Space, Button, Row, Col, Statistic, Modal, Input,
  notification, Alert, Popconfirm, Tooltip, Tabs,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SafetyCertificateOutlined, PlusOutlined, ReloadOutlined, DeleteOutlined,
  LinkOutlined, DisconnectOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';
import ExportButton from '../../components/ExportButton';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

interface GPO {
  id: string;
  name: string;
  dn: string;
  path: string;
  linksTo: string[] | null;
  version: number;
  flags: number;
}

interface GPOLink {
  gpoId: string;
  ouDn: string;
  enabled: boolean;
}

export default function GPOPage() {
  const [gpos, setGpos] = useState<GPO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [newGPOName, setNewGPOName] = useState('');
  const [creating, setCreating] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkGPO, setLinkGPO] = useState<GPO | null>(null);
  const [linkOU, setLinkOU] = useState('');
  const [linking, setLinking] = useState(false);
  const [selectedGPO, setSelectedGPO] = useState<GPO | null>(null);
  const [links, setLinks] = useState<GPOLink[]>([]);
  const [linksLoading, setLinksLoading] = useState(false);

  const fetchGPOs = () => {
    setLoading(true);
    setError(null);
    api.get<{ gpos: GPO[]; total: number }>('/gpo')
      .then((data) => setGpos(data.gpos || []))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchGPOs(); }, []);

  const handleCreate = async () => {
    if (!newGPOName.trim()) return;
    setCreating(true);
    try {
      await api.post('/gpo', { name: newGPOName.trim() });
      notification.success({
        message: 'GPO created',
        description: `GPO "${newGPOName}" created successfully`,
      });
      setCreateOpen(false);
      setNewGPOName('');
      fetchGPOs();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Create failed';
      notification.error({ message: 'Create failed', description: msg });
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (gpo: GPO) => {
    try {
      await api.delete(`/gpo/${encodeURIComponent(gpo.id)}`);
      notification.success({
        message: 'GPO deleted',
        description: `GPO "${gpo.name}" deleted`,
      });
      fetchGPOs();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Delete failed';
      notification.error({ message: 'Delete failed', description: msg });
    }
  };

  const openLinkModal = (gpo: GPO) => {
    setLinkGPO(gpo);
    setLinkOU('');
    setLinkOpen(true);
  };

  const handleLink = async () => {
    if (!linkGPO || !linkOU.trim()) return;
    setLinking(true);
    try {
      await api.post(`/gpo/${encodeURIComponent(linkGPO.id)}/link`, { ouDn: linkOU.trim() });
      notification.success({
        message: 'GPO linked',
        description: `GPO "${linkGPO.name}" linked to ${linkOU}`,
      });
      setLinkOpen(false);
      fetchGPOs();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Link failed';
      notification.error({ message: 'Link failed', description: msg });
    } finally {
      setLinking(false);
    }
  };

  const handleUnlink = async (gpoId: string, ouDn: string) => {
    try {
      await api.delete(`/gpo/${encodeURIComponent(gpoId)}/link`, { ouDn });
      notification.success({ message: 'GPO unlinked' });
      if (selectedGPO) {
        fetchGPOLinks(selectedGPO);
      }
      fetchGPOs();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Unlink failed';
      notification.error({ message: 'Unlink failed', description: msg });
    }
  };

  const fetchGPOLinks = (gpo: GPO) => {
    setSelectedGPO(gpo);
    setLinksLoading(true);
    // Use the GPO detail endpoint
    api.get<GPO>(`/gpo/${encodeURIComponent(gpo.id)}`)
      .then((data) => {
        // Build links from GPO detail or try getlink
        setLinks([]);
        setSelectedGPO(data);
      })
      .catch(() => setLinks([]))
      .finally(() => setLinksLoading(false));
  };

  const flagsLabel = (flags: number) => {
    switch (flags) {
      case 0: return <Tag color="green">Enabled</Tag>;
      case 1: return <Tag color="orange">User settings disabled</Tag>;
      case 2: return <Tag color="orange">Computer settings disabled</Tag>;
      case 3: return <Tag color="red">All settings disabled</Tag>;
      default: return <Tag>{flags}</Tag>;
    }
  };

  const columns: ColumnsType<GPO> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: GPO) => (
        <Space>
          <SafetyCertificateOutlined />
          <a onClick={() => fetchGPOLinks(record)} style={mono}>{name}</a>
        </Space>
      ),
    },
    {
      title: 'GUID',
      dataIndex: 'id',
      key: 'id',
      width: 320,
      render: (id: string) => (
        <Text copyable style={{ ...mono, fontSize: 12 }}>{id}</Text>
      ),
    },
    {
      title: 'Version',
      dataIndex: 'version',
      key: 'version',
      width: 100,
      align: 'center',
    },
    {
      title: 'Status',
      dataIndex: 'flags',
      key: 'flags',
      width: 180,
      render: (flags: number) => flagsLabel(flags),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 140,
      render: (_: unknown, record: GPO) => (
        <Space>
          <Tooltip title="Link to OU">
            <Button size="small" icon={<LinkOutlined />} onClick={() => openLinkModal(record)} />
          </Tooltip>
          <Popconfirm
            title={`Delete GPO "${record.name}"?`}
            description="This action cannot be undone."
            onConfirm={() => handleDelete(record)}
            okType="danger"
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const linkColumns: ColumnsType<GPOLink> = [
    {
      title: 'GPO',
      dataIndex: 'gpoId',
      key: 'gpoId',
      render: (id: string) => <Text style={{ ...mono, fontSize: 12 }}>{id}</Text>,
    },
    {
      title: 'Linked OU',
      dataIndex: 'ouDn',
      key: 'ouDn',
      render: (dn: string) => <Text style={{ ...mono, fontSize: 12 }}>{dn}</Text>,
    },
    {
      title: '',
      key: 'actions',
      width: 80,
      render: (_: unknown, record: GPOLink) => (
        <Popconfirm
          title="Remove this GPO link?"
          onConfirm={() => handleUnlink(record.gpoId, record.ouDn)}
        >
          <Button size="small" danger icon={<DisconnectOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <SafetyCertificateOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Group Policy Objects</Title>
        </Space>
        <Space>
          <ExportButton
            data={gpos as unknown as Record<string, unknown>[]}
            filename="sambmin-gpos"
            columns={[
              { key: 'name', title: 'Name' },
              { key: 'id', title: 'GUID' },
              { key: 'version', title: 'Version' },
              { key: 'flags', title: 'Flags' },
              { key: 'dn', title: 'DN' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchGPOs}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            New GPO
          </Button>
        </Space>
      </div>

      {error && (
        <Alert type="error" message="Failed to load GPOs" description={error} style={{ marginBottom: 16 }} />
      )}

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card size="small">
            <Statistic title="Total GPOs" value={gpos.length} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="Enabled"
              value={gpos.filter((g) => g.flags === 0).length}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="Disabled"
              value={gpos.filter((g) => g.flags > 0).length}
              valueStyle={{ color: gpos.some((g) => g.flags > 0) ? '#faad14' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      <Tabs
        items={[
          {
            key: 'gpos',
            label: 'GPO List',
            children: (
              <Table
                columns={columns}
                dataSource={gpos}
                rowKey="id"
                loading={loading}
                pagination={false}
                size="middle"
              />
            ),
          },
          {
            key: 'detail',
            label: selectedGPO ? `Detail: ${selectedGPO.name}` : 'Detail',
            disabled: !selectedGPO,
            children: selectedGPO ? (
              <Card
                title={selectedGPO.name}
                loading={linksLoading}
                extra={<Button size="small" onClick={() => setSelectedGPO(null)}>Close</Button>}
              >
                <Space direction="vertical" style={{ width: '100%' }} size={12}>
                  <Row gutter={16}>
                    <Col span={12}>
                      <Text type="secondary">GUID</Text>
                      <br />
                      <Text copyable style={mono}>{selectedGPO.id}</Text>
                    </Col>
                    <Col span={12}>
                      <Text type="secondary">Version</Text>
                      <br />
                      <Text style={mono}>{selectedGPO.version}</Text>
                    </Col>
                  </Row>
                  <div>
                    <Text type="secondary">DN</Text>
                    <br />
                    <Text copyable style={{ ...mono, fontSize: 12 }}>{selectedGPO.dn}</Text>
                  </div>
                  <div>
                    <Text type="secondary">Path</Text>
                    <br />
                    <Text copyable style={{ ...mono, fontSize: 12 }}>{selectedGPO.path}</Text>
                  </div>
                  <div>
                    <Text type="secondary">Status</Text>
                    <br />
                    {flagsLabel(selectedGPO.flags)}
                  </div>
                  {links.length > 0 && (
                    <div>
                      <Text type="secondary">Links</Text>
                      <Table
                        columns={linkColumns}
                        dataSource={links}
                        rowKey={(r) => `${r.gpoId}-${r.ouDn}`}
                        pagination={false}
                        size="small"
                        style={{ marginTop: 8 }}
                      />
                    </div>
                  )}
                </Space>
              </Card>
            ) : null,
          },
        ]}
      />

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space direction="vertical" size={4}>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool gpo listall
          </Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool gpo show {'<GPO-GUID>'}
          </Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool gpo setlink {'<OU-DN>'} {'<GPO-GUID>'}
          </Text>
        </Space>
      </Card>

      {/* Create GPO Modal */}
      <Modal
        title="Create Group Policy Object"
        open={createOpen}
        onCancel={() => { setCreateOpen(false); setNewGPOName(''); }}
        onOk={handleCreate}
        okText="Create"
        confirmLoading={creating}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>GPO Name:</Text>
            <Input
              value={newGPOName}
              onChange={(e) => setNewGPOName(e.target.value)}
              placeholder="e.g. Password Policy - IT Staff"
              style={mono}
            />
          </div>
          <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
            <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
            <Text copyable style={{ ...mono, fontSize: 12 }}>
              samba-tool gpo create {newGPOName || '<name>'}
            </Text>
          </Card>
        </Space>
      </Modal>

      {/* Link GPO Modal */}
      <Modal
        title={`Link GPO: ${linkGPO?.name || ''}`}
        open={linkOpen}
        onCancel={() => { setLinkOpen(false); setLinkOU(''); }}
        onOk={handleLink}
        okText="Link"
        confirmLoading={linking}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>Target OU (Distinguished Name):</Text>
            <Input
              value={linkOU}
              onChange={(e) => setLinkOU(e.target.value)}
              placeholder="e.g. OU=Staff,DC=example,DC=com"
              style={mono}
            />
          </div>
          <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
            <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
            <Text copyable style={{ ...mono, fontSize: 12 }}>
              samba-tool gpo setlink {linkOU || '<OU-DN>'} {linkGPO?.id || '<GPO-GUID>'}
            </Text>
          </Card>
        </Space>
      </Modal>
    </div>
  );
}
