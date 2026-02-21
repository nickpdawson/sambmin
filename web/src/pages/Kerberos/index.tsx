import { useState, useEffect } from 'react';
import {
  Typography, Table, Card, Space, Button, Row, Col, Input, notification, Alert,
  Tabs, Popconfirm, Tag, Descriptions, Spin, Badge, Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  KeyOutlined, SearchOutlined, PlusOutlined, DeleteOutlined,
  SafetyCertificateOutlined, TeamOutlined, DownloadOutlined,
  DesktopOutlined, UserOutlined, ReloadOutlined,
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

interface KerberosPolicy {
  realm: string;
  baseDN: string;
  maxPwdAge: string;
  minPwdAge: string;
  minPwdLength: string;
  pwdHistoryLength: string;
  lockoutDuration: string;
  lockoutThreshold: string;
  lockoutObservationWindow: string;
  supportedEncryptionTypes: string;
  implementation: string;
  kdc: string;
  keytabConfigured: boolean;
}

interface KerberosAccount {
  dn: string;
  samAccountName: string;
  displayName: string;
  objectType: string;
  spns: string[];
  spnCount: number;
  encryptionTypes: string;
}

// Convert AD 100-nanosecond interval to human-readable duration
function formatADInterval(raw: string): string {
  if (!raw) return 'Not set';
  const val = BigInt(raw);
  if (val === 0n) return 'None';
  // AD stores as negative 100-nanosecond intervals
  const positive = val < 0n ? -val : val;
  const seconds = Number(positive / 10000000n);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) return `${days} day${days !== 1 ? 's' : ''}`;
  if (hours > 0) return `${hours} hour${hours !== 1 ? 's' : ''}`;
  if (minutes > 0) return `${minutes} minute${minutes !== 1 ? 's' : ''}`;
  return `${seconds} second${seconds !== 1 ? 's' : ''}`;
}

// Decode encryption type bitmask
function decodeEncryptionTypes(raw: string): string[] {
  if (!raw) return ['Default (RC4)'];
  const val = parseInt(raw, 10);
  if (isNaN(val)) return ['Unknown'];
  const types: string[] = [];
  if (val & 0x1) types.push('DES-CBC-CRC');
  if (val & 0x2) types.push('DES-CBC-MD5');
  if (val & 0x4) types.push('RC4-HMAC');
  if (val & 0x8) types.push('AES128-CTS');
  if (val & 0x10) types.push('AES256-CTS');
  return types.length > 0 ? types : ['None configured'];
}

export default function Kerberos() {
  // Policy state
  const [policy, setPolicy] = useState<KerberosPolicy | null>(null);
  const [policyLoading, setPolicyLoading] = useState(false);
  const [policyError, setPolicyError] = useState<string | null>(null);

  // Accounts state
  const [accounts, setAccounts] = useState<KerberosAccount[]>([]);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [accountsError, setAccountsError] = useState<string | null>(null);
  const [accountSearch, setAccountSearch] = useState('');

  // Keytab export state
  const [keytabPrincipals, setKeytabPrincipals] = useState<string[]>(['']);
  const [keytabExporting, setKeytabExporting] = useState(false);
  const [keytabFallback, setKeytabFallback] = useState<{ message: string; commands: string[] } | null>(null);

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

  // Load policy on mount
  useEffect(() => {
    fetchPolicy();
  }, []);

  const fetchPolicy = () => {
    setPolicyLoading(true);
    setPolicyError(null);
    api.get<KerberosPolicy>('/kerberos/policy')
      .then((data) => setPolicy(data))
      .catch((err) => setPolicyError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setPolicyLoading(false));
  };

  const fetchAccounts = () => {
    setAccountsLoading(true);
    setAccountsError(null);
    api.get<{ accounts: KerberosAccount[]; total: number }>('/kerberos/accounts')
      .then((data) => setAccounts(data.accounts || []))
      .catch((err) => setAccountsError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setAccountsLoading(false));
  };

  const handleExportKeytab = async () => {
    const principals = keytabPrincipals.map((p) => p.trim()).filter(Boolean);
    if (principals.length === 0) return;
    setKeytabExporting(true);
    setKeytabFallback(null);
    try {
      const response = await fetch(`/api/kerberos/keytab`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ principals }),
      });

      const contentType = response.headers.get('content-type') || '';

      // Check if the response is JSON (could be cli_fallback or error)
      if (contentType.includes('application/json')) {
        const data = await response.json();
        if (data.mode === 'cli_fallback') {
          setKeytabFallback({ message: data.message, commands: data.commands });
          return;
        }
        if (data.error) {
          throw new Error(data.error);
        }
      }

      if (!response.ok) {
        throw new Error(response.statusText);
      }

      // Binary download
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = principals.length === 1
        ? `${principals[0].replace(/\//g, '_')}.keytab`
        : 'service.keytab';
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
      notification.success({
        message: 'Keytab exported',
        description: `Downloaded keytab with ${principals.length} principal${principals.length > 1 ? 's' : ''}`,
      });
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Export failed';
      notification.error({ message: 'Keytab export failed', description: msg });
    } finally {
      setKeytabExporting(false);
    }
  };

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

  const filteredAccounts = accountSearch
    ? accounts.filter((a) =>
        a.samAccountName.toLowerCase().includes(accountSearch.toLowerCase()) ||
        a.displayName.toLowerCase().includes(accountSearch.toLowerCase()) ||
        a.spns.some((s) => s.toLowerCase().includes(accountSearch.toLowerCase()))
      )
    : accounts;

  const accountColumns: ColumnsType<KerberosAccount> = [
    {
      title: 'Account',
      dataIndex: 'samAccountName',
      key: 'samAccountName',
      sorter: (a, b) => a.samAccountName.localeCompare(b.samAccountName),
      render: (val: string, record) => (
        <Space>
          {record.objectType === 'computer' ? (
            <DesktopOutlined style={{ color: '#722ed1' }} />
          ) : (
            <UserOutlined style={{ color: '#1677ff' }} />
          )}
          <Text style={mono}>{val}</Text>
        </Space>
      ),
    },
    {
      title: 'Display Name',
      dataIndex: 'displayName',
      key: 'displayName',
      ellipsis: true,
    },
    {
      title: 'Type',
      dataIndex: 'objectType',
      key: 'objectType',
      width: 100,
      render: (val: string) => (
        <Tag color={val === 'computer' ? 'purple' : 'blue'}>{val}</Tag>
      ),
      filters: [
        { text: 'User', value: 'user' },
        { text: 'Computer', value: 'computer' },
      ],
      onFilter: (value, record) => record.objectType === value,
    },
    {
      title: 'SPNs',
      dataIndex: 'spnCount',
      key: 'spnCount',
      width: 80,
      sorter: (a, b) => a.spnCount - b.spnCount,
      render: (val: number) => <Badge count={val} style={{ backgroundColor: '#52c41a' }} />,
    },
    {
      title: 'Encryption',
      dataIndex: 'encryptionTypes',
      key: 'encryptionTypes',
      width: 200,
      render: (val: string) => (
        <Space size={2} wrap>
          {decodeEncryptionTypes(val).map((t) => (
            <Tag key={t} color={t.startsWith('AES') ? 'green' : t.startsWith('DES') ? 'red' : 'default'}>
              {t}
            </Tag>
          ))}
        </Space>
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
          key: 'policy',
          label: (
            <span><SafetyCertificateOutlined /> Policy</span>
          ),
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              {policyLoading && <Spin tip="Loading Kerberos policy..." />}
              {policyError && (
                <Alert
                  type="error"
                  message="Failed to load policy"
                  description={policyError}
                  action={<Button size="small" onClick={fetchPolicy}>Retry</Button>}
                />
              )}
              {policy && (
                <>
                  <Card title="Realm Configuration" size="small">
                    <Descriptions bordered size="small" column={{ xs: 1, sm: 2 }}>
                      <Descriptions.Item label="Realm">
                        <Text strong style={mono}>{policy.realm}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="Base DN">
                        <Text style={mono}>{policy.baseDN}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="Implementation">
                        <Tag color="blue">{policy.implementation || 'Unknown'}</Tag>
                      </Descriptions.Item>
                      <Descriptions.Item label="KDC">
                        <Text style={mono}>{policy.kdc || 'Not configured'}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="Keytab">
                        {policy.keytabConfigured ? (
                          <Tag color="green">Configured</Tag>
                        ) : (
                          <Tag>Not configured</Tag>
                        )}
                      </Descriptions.Item>
                    </Descriptions>
                  </Card>

                  <Card title="Supported Encryption Types" size="small">
                    <Space size={8} wrap>
                      {decodeEncryptionTypes(policy.supportedEncryptionTypes).map((t) => (
                        <Tag
                          key={t}
                          color={t.startsWith('AES') ? 'green' : t.startsWith('DES') ? 'red' : 'default'}
                          style={{ fontSize: 13, padding: '4px 12px' }}
                        >
                          {t}
                        </Tag>
                      ))}
                    </Space>
                  </Card>

                  <Row gutter={16}>
                    <Col xs={24} md={12}>
                      <Card title="Password Policy" size="small">
                        <Descriptions bordered size="small" column={1}>
                          <Descriptions.Item label="Max Password Age">
                            {formatADInterval(policy.maxPwdAge)}
                          </Descriptions.Item>
                          <Descriptions.Item label="Min Password Age">
                            {formatADInterval(policy.minPwdAge)}
                          </Descriptions.Item>
                          <Descriptions.Item label="Min Password Length">
                            {policy.minPwdLength || 'Not set'} characters
                          </Descriptions.Item>
                          <Descriptions.Item label="Password History">
                            {policy.pwdHistoryLength || '0'} passwords remembered
                          </Descriptions.Item>
                        </Descriptions>
                      </Card>
                    </Col>
                    <Col xs={24} md={12}>
                      <Card title="Lockout Policy" size="small">
                        <Descriptions bordered size="small" column={1}>
                          <Descriptions.Item label="Lockout Threshold">
                            {policy.lockoutThreshold === '0' ? 'Disabled' : `${policy.lockoutThreshold} attempts`}
                          </Descriptions.Item>
                          <Descriptions.Item label="Lockout Duration">
                            {formatADInterval(policy.lockoutDuration)}
                          </Descriptions.Item>
                          <Descriptions.Item label="Observation Window">
                            {formatADInterval(policy.lockoutObservationWindow)}
                          </Descriptions.Item>
                        </Descriptions>
                      </Card>
                    </Col>
                  </Row>
                </>
              )}
            </Space>
          ),
        },
        {
          key: 'accounts',
          label: (
            <span><TeamOutlined /> Accounts</span>
          ),
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Card size="small">
                <Row justify="space-between" align="middle">
                  <Col>
                    <Text type="secondary">
                      Accounts with registered Service Principal Names
                    </Text>
                  </Col>
                  <Col>
                    <Space>
                      <Input
                        placeholder="Filter accounts..."
                        prefix={<SearchOutlined />}
                        value={accountSearch}
                        onChange={(e) => setAccountSearch(e.target.value)}
                        style={{ width: 240 }}
                        allowClear
                      />
                      <Button
                        icon={<ReloadOutlined />}
                        onClick={fetchAccounts}
                        loading={accountsLoading}
                      >
                        {accounts.length === 0 ? 'Load' : 'Refresh'}
                      </Button>
                    </Space>
                  </Col>
                </Row>
              </Card>

              {accountsError && (
                <Alert type="error" message="Failed to load accounts" description={accountsError} />
              )}

              {accounts.length > 0 && (
                <Table
                  columns={accountColumns}
                  dataSource={filteredAccounts}
                  rowKey="dn"
                  size="small"
                  pagination={{ pageSize: 25, showSizeChanger: true, showTotal: (t) => `${t} accounts` }}
                  expandable={{
                    expandedRowRender: (record) => (
                      <Space direction="vertical" size={4} style={{ padding: '8px 0' }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>DN: </Text>
                        <Text style={{ ...mono, fontSize: 12 }}>{record.dn}</Text>
                        <Text type="secondary" style={{ fontSize: 12, marginTop: 8, display: 'block' }}>SPNs:</Text>
                        {record.spns.map((spn) => (
                          <Text key={spn} copyable style={{ ...mono, fontSize: 12, display: 'block' }}>
                            {spn}
                          </Text>
                        ))}
                      </Space>
                    ),
                  }}
                />
              )}

              {accounts.length === 0 && !accountsLoading && !accountsError && (
                <Alert
                  type="info"
                  message="Click Load to fetch accounts with SPNs"
                  description="This queries all accounts in the domain that have registered Service Principal Names."
                />
              )}
            </Space>
          ),
        },
        {
          key: 'keytab',
          label: (
            <span><DownloadOutlined /> Keytab Export</span>
          ),
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <Alert
                type="info"
                message="Keytab Export"
                description="Export a Kerberos keytab file containing one or more service principals. Multiple principals can be bundled into a single keytab. Requires Domain Admin authentication."
              />

              <Card title="Export Keytab" size="small">
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
                      Principals to include in keytab:
                    </Text>
                    {keytabPrincipals.map((p, i) => (
                      <Space.Compact key={i} style={{ width: '100%', marginBottom: 4 }}>
                        <Input
                          placeholder="e.g. HTTP/web.dzsec.net"
                          value={p}
                          onChange={(e) => {
                            const updated = [...keytabPrincipals];
                            updated[i] = e.target.value;
                            setKeytabPrincipals(updated);
                          }}
                          onPressEnter={() => {
                            if (i === keytabPrincipals.length - 1 && p.trim()) {
                              setKeytabPrincipals([...keytabPrincipals, '']);
                            }
                          }}
                          style={mono}
                        />
                        {keytabPrincipals.length > 1 && (
                          <Button
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => setKeytabPrincipals(keytabPrincipals.filter((_, j) => j !== i))}
                          />
                        )}
                      </Space.Compact>
                    ))}
                    <Space>
                      <Button
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={() => setKeytabPrincipals([...keytabPrincipals, ''])}
                      >
                        Add Principal
                      </Button>
                      <Button
                        type="primary"
                        icon={<DownloadOutlined />}
                        onClick={handleExportKeytab}
                        loading={keytabExporting}
                        disabled={!keytabPrincipals.some((p) => p.trim())}
                      >
                        Export Keytab ({keytabPrincipals.filter((p) => p.trim()).length})
                      </Button>
                    </Space>
                  </div>

                  <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
                    {keytabPrincipals.filter((p) => p.trim()).map((p, i) => (
                      <Text key={i} copyable style={{ ...mono, fontSize: 12, display: 'block' }}>
                        samba-tool domain exportkeytab /tmp/service.keytab --principal={p.trim()}
                      </Text>
                    ))}
                    {!keytabPrincipals.some((p) => p.trim()) && (
                      <Text style={{ ...mono, fontSize: 12 }}>
                        samba-tool domain exportkeytab /tmp/service.keytab --principal={'<principal>'}
                      </Text>
                    )}
                  </Card>

                  <Card size="small" title="Quick Add from Known SPNs">
                    <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 8 }}>
                      Click to add a principal to the export list:
                    </Text>
                    <Space direction="vertical" size={4} style={{ width: '100%' }}>
                      {[
                        { principal: 'HTTP/web.dzsec.net', desc: 'Web server authentication' },
                        { principal: 'cifs/fileserver.dzsec.net', desc: 'SMB/CIFS file sharing' },
                        { principal: 'ldap/dc.dzsec.net', desc: 'LDAP service' },
                        { principal: 'host/server.dzsec.net', desc: 'Host authentication' },
                      ].map((item) => (
                        <Row key={item.principal} justify="space-between" align="middle">
                          <Col>
                            <Space>
                              <Text style={mono}>{item.principal}</Text>
                              <Text type="secondary" style={{ fontSize: 12 }}>{item.desc}</Text>
                            </Space>
                          </Col>
                          <Col>
                            <Button
                              size="small"
                              type="link"
                              onClick={() => {
                                const emptyIdx = keytabPrincipals.findIndex((p) => !p.trim());
                                if (emptyIdx >= 0) {
                                  const updated = [...keytabPrincipals];
                                  updated[emptyIdx] = item.principal;
                                  setKeytabPrincipals(updated);
                                } else {
                                  setKeytabPrincipals([...keytabPrincipals, item.principal]);
                                }
                              }}
                            >
                              + Add
                            </Button>
                          </Col>
                        </Row>
                      ))}
                    </Space>
                  </Card>
                </Space>
              </Card>

              {keytabFallback && (
                <Card
                  title="Run on your Domain Controller"
                  size="small"
                  style={{ borderColor: '#faad14' }}
                  styles={{ header: { backgroundColor: 'rgba(250, 173, 20, 0.1)' } }}
                >
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <Text>{keytabFallback.message}</Text>
                    <div style={{
                      background: 'var(--ant-color-bg-container)',
                      border: '1px solid var(--ant-color-border)',
                      borderRadius: 6,
                      padding: 12,
                    }}>
                      {keytabFallback.commands.map((cmd, i) => (
                        <Text
                          key={i}
                          copyable
                          style={{ ...mono, display: 'block', marginBottom: i < keytabFallback.commands.length - 1 ? 4 : 0 }}
                        >
                          {cmd}
                        </Text>
                      ))}
                    </div>
                    <Alert
                      type="info"
                      showIcon
                      message="To enable in-app export, run sambmin as a user with read access to the Samba private directory."
                      style={{ fontSize: 12 }}
                    />
                  </Space>
                </Card>
              )}
            </Space>
          ),
        },
        {
          key: 'spn',
          label: 'SPN Management',
          children: (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
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
