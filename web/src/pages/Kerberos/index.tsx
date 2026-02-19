import { useState } from 'react';
import {
  Typography, Table, Card, Space, Button, Row, Col, Input, notification, Alert,
  Tabs, Popconfirm, Tag, Descriptions,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  KeyOutlined, SearchOutlined, PlusOutlined, DeleteOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

interface SPN {
  value: string;
  account: string;
}

interface DelegationInfo {
  account: string;
  unconstrained: boolean;
  constrained: boolean;
  allowedServices: string[];
}

export default function Kerberos() {
  // SPN state
  const [spnAccount, setSpnAccount] = useState('');
  const [spns, setSpns] = useState<SPN[]>([]);
  const [spnLoading, setSpnLoading] = useState(false);
  const [spnError, setSpnError] = useState<string | null>(null);
  const [addSPNOpen, setAddSPNOpen] = useState(false);
  const [newSPN, setNewSPN] = useState('');
  const [newSPNAccount, setNewSPNAccount] = useState('');
  const [addingSPN, setAddingSPN] = useState(false);

  // Delegation state
  const [delAccount, setDelAccount] = useState('');
  const [delegation, setDelegation] = useState<DelegationInfo | null>(null);
  const [delLoading, setDelLoading] = useState(false);
  const [delError, setDelError] = useState<string | null>(null);
  const [newService, setNewService] = useState('');
  const [addingService, setAddingService] = useState(false);

  const fetchSPNs = () => {
    if (!spnAccount.trim()) return;
    setSpnLoading(true);
    setSpnError(null);
    api.get<{ spns: SPN[]; total: number }>(`/spn/${encodeURIComponent(spnAccount.trim())}`)
      .then((data) => setSpns(data.spns || []))
      .catch((err) => {
        setSpnError(err instanceof Error ? err.message : 'Failed to load');
        setSpns([]);
      })
      .finally(() => setSpnLoading(false));
  };

  const handleAddSPN = async () => {
    if (!newSPN.trim() || !newSPNAccount.trim()) return;
    setAddingSPN(true);
    try {
      await api.post('/spn', { spn: newSPN.trim(), account: newSPNAccount.trim() });
      notification.success({
        message: 'SPN added',
        description: `${newSPN} added to ${newSPNAccount}`,
      });
      setAddSPNOpen(false);
      setNewSPN('');
      // Refresh if viewing the same account
      if (spnAccount.trim() === newSPNAccount.trim()) {
        fetchSPNs();
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Add failed';
      notification.error({ message: 'Add SPN failed', description: msg });
    } finally {
      setAddingSPN(false);
    }
  };

  const handleDeleteSPN = async (spn: SPN) => {
    try {
      await api.delete('/spn', { spn: spn.value, account: spn.account });
      notification.success({
        message: 'SPN deleted',
        description: `${spn.value} removed from ${spn.account}`,
      });
      fetchSPNs();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Delete failed';
      notification.error({ message: 'Delete SPN failed', description: msg });
    }
  };

  const fetchDelegation = () => {
    if (!delAccount.trim()) return;
    setDelLoading(true);
    setDelError(null);
    api.get<DelegationInfo>(`/delegation/${encodeURIComponent(delAccount.trim())}`)
      .then((data) => setDelegation(data))
      .catch((err) => {
        setDelError(err instanceof Error ? err.message : 'Failed to load');
        setDelegation(null);
      })
      .finally(() => setDelLoading(false));
  };

  const handleAddService = async () => {
    if (!delegation || !newService.trim()) return;
    setAddingService(true);
    try {
      await api.post(`/delegation/${encodeURIComponent(delegation.account)}/service`, {
        service: newService.trim(),
      });
      notification.success({
        message: 'Delegation service added',
        description: `${newService} added to ${delegation.account}`,
      });
      setNewService('');
      fetchDelegation();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Add failed';
      notification.error({ message: 'Add service failed', description: msg });
    } finally {
      setAddingService(false);
    }
  };

  const handleRemoveService = async (service: string) => {
    if (!delegation) return;
    try {
      await api.delete(`/delegation/${encodeURIComponent(delegation.account)}/service`, {
        service,
      });
      notification.success({ message: 'Service removed' });
      fetchDelegation();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Remove failed';
      notification.error({ message: 'Remove failed', description: msg });
    }
  };

  const spnColumns: ColumnsType<SPN> = [
    {
      title: 'Service Principal Name',
      dataIndex: 'value',
      key: 'value',
      render: (val: string) => <Text copyable style={mono}>{val}</Text>,
    },
    {
      title: 'Service',
      key: 'service',
      width: 120,
      render: (_: unknown, record: SPN) => {
        const svc = record.value.split('/')[0];
        return <Tag>{svc}</Tag>;
      },
    },
    {
      title: '',
      key: 'actions',
      width: 60,
      render: (_: unknown, record: SPN) => (
        <Popconfirm
          title={`Delete SPN "${record.value}"?`}
          onConfirm={() => handleDeleteSPN(record)}
          okType="danger"
        >
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <KeyOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Kerberos & SPNs</Title>
        </Space>
      </div>

      <Tabs items={[
        {
          key: 'spn',
          label: 'Service Principal Names',
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              {/* SPN Lookup */}
              <Card size="small" title="SPN Lookup">
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    placeholder="Enter account name (e.g. myhost$ or svc-account)"
                    value={spnAccount}
                    onChange={(e) => setSpnAccount(e.target.value)}
                    onPressEnter={fetchSPNs}
                    style={mono}
                  />
                  <Button
                    type="primary"
                    icon={<SearchOutlined />}
                    onClick={fetchSPNs}
                    loading={spnLoading}
                  >
                    Lookup
                  </Button>
                </Space.Compact>
              </Card>

              {spnError && (
                <Alert type="error" message="SPN lookup failed" description={spnError} />
              )}

              {spns.length > 0 && (
                <Card
                  title={`SPNs for ${spnAccount}`}
                  extra={
                    <Button
                      size="small"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        setNewSPNAccount(spnAccount);
                        setAddSPNOpen(true);
                      }}
                    >
                      Add SPN
                    </Button>
                  }
                >
                  <Table
                    columns={spnColumns}
                    dataSource={spns}
                    rowKey="value"
                    pagination={false}
                    size="small"
                  />
                </Card>
              )}

              {spnAccount && !spnLoading && !spnError && spns.length === 0 && (
                <Alert
                  type="info"
                  message="No SPNs found"
                  description={`No service principal names found for "${spnAccount}".`}
                  action={
                    <Button
                      size="small"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        setNewSPNAccount(spnAccount);
                        setAddSPNOpen(true);
                      }}
                    >
                      Add one
                    </Button>
                  }
                />
              )}

              {/* Add SPN inline form */}
              {addSPNOpen && (
                <Card title="Add Service Principal Name" size="small">
                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>Account:</Text>
                      <Input
                        value={newSPNAccount}
                        onChange={(e) => setNewSPNAccount(e.target.value)}
                        style={mono}
                        placeholder="e.g. myhost$"
                      />
                    </div>
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>SPN:</Text>
                      <Input
                        value={newSPN}
                        onChange={(e) => setNewSPN(e.target.value)}
                        style={mono}
                        placeholder="e.g. HTTP/myhost.dzsec.net"
                      />
                    </div>
                    <Space>
                      <Button type="primary" onClick={handleAddSPN} loading={addingSPN}>Add</Button>
                      <Button onClick={() => { setAddSPNOpen(false); setNewSPN(''); }}>Cancel</Button>
                    </Space>
                    <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
                      <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
                      <Text copyable style={{ ...mono, fontSize: 12 }}>
                        samba-tool spn add {newSPN || '<SPN>'} {newSPNAccount || '<account>'}
                      </Text>
                    </Card>
                  </Space>
                </Card>
              )}

              {/* CLI reference */}
              <Card size="small">
                <Space direction="vertical" size={4}>
                  <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
                  <Text copyable style={{ ...mono, fontSize: 12 }}>
                    samba-tool spn list {'<account>'}
                  </Text>
                  <Text copyable style={{ ...mono, fontSize: 12 }}>
                    samba-tool spn add {'<SPN>'} {'<account>'}
                  </Text>
                  <Text copyable style={{ ...mono, fontSize: 12 }}>
                    samba-tool spn delete {'<SPN>'} {'<account>'}
                  </Text>
                </Space>
              </Card>
            </Space>
          ),
        },
        {
          key: 'delegation',
          label: 'Delegation',
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              {/* Delegation Lookup */}
              <Card size="small" title="Delegation Lookup">
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    placeholder="Enter account name (e.g. svc-sql$)"
                    value={delAccount}
                    onChange={(e) => setDelAccount(e.target.value)}
                    onPressEnter={fetchDelegation}
                    style={mono}
                  />
                  <Button
                    type="primary"
                    icon={<SearchOutlined />}
                    onClick={fetchDelegation}
                    loading={delLoading}
                  >
                    Lookup
                  </Button>
                </Space.Compact>
              </Card>

              {delError && (
                <Alert type="error" message="Delegation lookup failed" description={delError} />
              )}

              {delegation && (
                <Card title={`Delegation: ${delegation.account}`}>
                  <Space direction="vertical" size={16} style={{ width: '100%' }}>
                    <Descriptions bordered size="small" column={1}>
                      <Descriptions.Item label="Account">
                        <Text style={mono}>{delegation.account}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="Unconstrained">
                        {delegation.unconstrained ? (
                          <Tag color="red">Trusted for delegation (any service)</Tag>
                        ) : (
                          <Tag>No</Tag>
                        )}
                      </Descriptions.Item>
                      <Descriptions.Item label="Constrained">
                        {delegation.constrained ? (
                          <Tag color="blue">Constrained delegation</Tag>
                        ) : (
                          <Tag>No</Tag>
                        )}
                      </Descriptions.Item>
                    </Descriptions>

                    <Card
                      size="small"
                      title="Allowed Services"
                      extra={
                        <Space.Compact>
                          <Input
                            placeholder="cifs/server.dzsec.net"
                            value={newService}
                            onChange={(e) => setNewService(e.target.value)}
                            style={{ ...mono, width: 280 }}
                            size="small"
                          />
                          <Button
                            size="small"
                            type="primary"
                            icon={<PlusOutlined />}
                            onClick={handleAddService}
                            loading={addingService}
                          >
                            Add
                          </Button>
                        </Space.Compact>
                      }
                    >
                      {delegation.allowedServices.length > 0 ? (
                        <Space direction="vertical" size={4} style={{ width: '100%' }}>
                          {delegation.allowedServices.map((svc) => (
                            <Row key={svc} justify="space-between" align="middle">
                              <Col>
                                <Text copyable style={mono}>{svc}</Text>
                              </Col>
                              <Col>
                                <Popconfirm
                                  title={`Remove service "${svc}"?`}
                                  onConfirm={() => handleRemoveService(svc)}
                                  okType="danger"
                                >
                                  <Button size="small" danger icon={<DeleteOutlined />} />
                                </Popconfirm>
                              </Col>
                            </Row>
                          ))}
                        </Space>
                      ) : (
                        <Text type="secondary">No constrained delegation services configured</Text>
                      )}
                    </Card>
                  </Space>
                </Card>
              )}

              {/* CLI reference */}
              <Card size="small">
                <Space direction="vertical" size={4}>
                  <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
                  <Text copyable style={{ ...mono, fontSize: 12 }}>
                    samba-tool delegation show {'<account>'}
                  </Text>
                  <Text copyable style={{ ...mono, fontSize: 12 }}>
                    samba-tool delegation add-service {'<account>'} {'<service/host>'}
                  </Text>
                  <Text copyable style={{ ...mono, fontSize: 12 }}>
                    samba-tool delegation del-service {'<account>'} {'<service/host>'}
                  </Text>
                </Space>
              </Card>
            </Space>
          ),
        },
      ]} />
    </div>
  );
}
