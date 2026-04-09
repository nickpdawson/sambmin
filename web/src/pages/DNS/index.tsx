import { useState, useEffect, useMemo } from 'react';
import {
  Typography, Table, Tag, Badge, Tabs, Card, Space, Button, Input,
  Tooltip, Alert, Descriptions, Row, Col, Statistic, Dropdown, Modal,
  notification,
} from 'antd';
import {
  GlobalOutlined, PlusOutlined, ReloadOutlined, SearchOutlined,
  CheckCircleOutlined, WarningOutlined, ExclamationCircleOutlined,
  InfoCircleOutlined, CopyOutlined, ArrowLeftOutlined,
  EditOutlined, DeleteOutlined, MoreOutlined, LinkOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { MenuProps } from 'antd';
import { api } from '../../api/client';
import CreateRecordDrawer from './CreateRecordDrawer';
import ServerInfoTab from './ServerInfoTab';
import SRVValidatorTab from './SRVValidatorTab';
import ConsistencyTab from './ConsistencyTab';
import QueryToolTab from './QueryToolTab';
import ZonePropertiesPanel from './ZonePropertiesPanel';

const { Title, Text } = Typography;

interface DNSZone {
  name: string;
  type: string;
  backend: string;
  records: number;
  dynamic: boolean;
  soaSerial: number;
  status: string;
}

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

interface DiagnosticCheck {
  name: string;
  status: string;
  message: string;
}

const statusColors: Record<string, string> = {
  healthy: 'green',
  warning: 'orange',
  stale: 'red',
  error: 'red',
};

const statusBadge: Record<string, 'success' | 'warning' | 'error' | 'processing' | 'default'> = {
  healthy: 'success',
  warning: 'warning',
  stale: 'error',
  error: 'error',
};

const diagStatusIcon: Record<string, React.ReactNode> = {
  pass: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
  warning: <WarningOutlined style={{ color: '#faad14' }} />,
  fail: <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />,
  info: <InfoCircleOutlined style={{ color: '#1677ff' }} />,
};

const recordTypeTabs = [
  { key: 'all', label: 'All' },
  { key: 'A', label: 'A / AAAA' },
  { key: 'CNAME', label: 'CNAME' },
  { key: 'MX', label: 'MX' },
  { key: 'SRV', label: 'SRV' },
  { key: 'TXT', label: 'TXT' },
  { key: 'NS', label: 'NS' },
  { key: 'SOA', label: 'SOA' },
  { key: 'PTR', label: 'PTR' },
];

export default function DNS() {
  const [zones, setZones] = useState<DNSZone[]>([]);
  const [records, setRecords] = useState<DNSRecord[]>([]);
  const [diagnostics, setDiagnostics] = useState<DiagnosticCheck[]>([]);
  const [selectedZone, setSelectedZone] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState('zones');
  const [recordTypeFilter, setRecordTypeFilter] = useState('all');
  const [recordSearch, setRecordSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editRecord, setEditRecord] = useState<DNSRecord | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DNSRecord | null>(null);

  useEffect(() => {
    api.get<{ zones: DNSZone[] }>('/dns/zones')
      .then((data) => setZones(data.zones))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (selectedZone) {
      api.get<{ records: DNSRecord[] }>(`/dns/zones/${encodeURIComponent(selectedZone)}/records`)
        .then((data) => setRecords(data.records));
    }
  }, [selectedZone]);

  useEffect(() => {
    if (activeTab === 'diagnostics' && diagnostics.length === 0) {
      api.get<{ checks: DiagnosticCheck[] }>('/dns/diagnostics')
        .then((data) => setDiagnostics(data.checks));
    }
  }, [activeTab, diagnostics.length]);

  const filteredRecords = useMemo(() => {
    let filtered = records;
    if (recordTypeFilter !== 'all') {
      if (recordTypeFilter === 'A') {
        filtered = filtered.filter((r) => r.type === 'A' || r.type === 'AAAA');
      } else {
        filtered = filtered.filter((r) => r.type === recordTypeFilter);
      }
    }
    if (recordSearch) {
      const q = recordSearch.toLowerCase();
      filtered = filtered.filter(
        (r) => r.name.toLowerCase().includes(q) || r.value.toLowerCase().includes(q)
      );
    }
    return filtered;
  }, [records, recordTypeFilter, recordSearch]);

  // Zone summary stats
  const zoneStats = useMemo(() => {
    const totalRecords = zones.reduce((sum, z) => sum + z.records, 0);
    const sambaZones = zones.filter((z) => z.backend === 'samba').length;
    const bindZones = zones.filter((z) => z.backend === 'bind9').length;
    const warnings = zones.filter((z) => z.status !== 'healthy').length;
    return { totalRecords, sambaZones, bindZones, warnings };
  }, [zones]);

  const zoneColumns: ColumnsType<DNSZone> = [
    {
      title: 'Zone',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: DNSZone) => (
        <Space>
          <Badge status={statusBadge[record.status] || 'default'} />
          <a onClick={() => { setSelectedZone(name); setActiveTab('records'); }}
             style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>
            {name}
          </a>
        </Space>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={type === 'forward' ? 'blue' : 'purple'}>{type}</Tag>
      ),
    },
    {
      title: 'Backend',
      dataIndex: 'backend',
      key: 'backend',
      width: 100,
      render: (backend: string) => (
        <Tag color={backend === 'samba' ? 'green' : 'orange'}>{backend}</Tag>
      ),
    },
    {
      title: 'Records',
      dataIndex: 'records',
      key: 'records',
      width: 90,
      align: 'right' as const,
      sorter: (a: DNSZone, b: DNSZone) => a.records - b.records,
    },
    {
      title: 'Dynamic',
      dataIndex: 'dynamic',
      key: 'dynamic',
      width: 90,
      render: (dynamic: boolean) => dynamic ? <Tag color="cyan">Yes</Tag> : <Tag>No</Tag>,
    },
    {
      title: 'SOA Serial',
      dataIndex: 'soaSerial',
      key: 'soaSerial',
      width: 130,
      render: (serial: number) => (
        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
          {serial}
        </Text>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>{status}</Tag>
      ),
    },
  ];

  // Lookup the currently selected zone object for the detail header
  const currentZone = useMemo(
    () => zones.find((z) => z.name === selectedZone) || null,
    [zones, selectedZone],
  );

  const getFQDN = (name: string) =>
    name === '@' ? selectedZone || '' : `${name}.${selectedZone}`;

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
  };

  const handleEditRecord = (record: DNSRecord) => {
    setEditRecord(record);
    setDrawerOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget || !selectedZone) return;
    try {
      const zoneEnc = encodeURIComponent(selectedZone);
      const nameEnc = encodeURIComponent(deleteTarget.name);
      const typeEnc = encodeURIComponent(deleteTarget.type);
      const valueEnc = encodeURIComponent(deleteTarget.value);
      await api.delete(`/dns/zones/${zoneEnc}/records/${nameEnc}?type=${typeEnc}&value=${valueEnc}`);
      notification.success({
        message: 'Record deleted',
        description: `${deleteTarget.type} ${deleteTarget.name} removed from ${selectedZone}`,
      });
      // Re-fetch records
      const data = await api.get<{ records: DNSRecord[] }>(`/dns/zones/${zoneEnc}/records`);
      setRecords(data.records);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Delete failed';
      notification.error({ message: 'Delete failed', description: msg });
    }
    setDeleteTarget(null);
  };

  const getRowActions = (record: DNSRecord): MenuProps['items'] => [
    {
      key: 'edit',
      icon: <EditOutlined />,
      label: 'Edit',
      onClick: () => handleEditRecord(record),
    },
    {
      key: 'copy-value',
      icon: <CopyOutlined />,
      label: 'Copy Value',
      onClick: () => handleCopy(record.value, 'Value'),
    },
    {
      key: 'copy-fqdn',
      icon: <LinkOutlined />,
      label: 'Copy Full FQDN',
      onClick: () => handleCopy(getFQDN(record.name), 'FQDN'),
    },
    { type: 'divider' },
    {
      key: 'delete',
      icon: <DeleteOutlined />,
      label: 'Delete',
      danger: true,
      onClick: () => setDeleteTarget(record),
    },
  ];

  const recordColumns: ColumnsType<DNSRecord> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (name: string) => (
        <Space>
          <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{name}</Text>
          <Tooltip title="Copy FQDN">
            <CopyOutlined
              style={{ color: 'var(--ant-color-text-tertiary)', cursor: 'pointer', fontSize: 12 }}
              onClick={() => handleCopy(getFQDN(name), 'FQDN')}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (type: string) => {
        const colors: Record<string, string> = {
          A: 'blue', AAAA: 'blue', CNAME: 'purple', MX: 'green',
          SRV: 'orange', TXT: 'cyan', NS: 'magenta', SOA: 'red', PTR: 'gold',
        };
        return <Tag color={colors[type] || 'default'}>{type}</Tag>;
      },
    },
    {
      title: 'Value',
      dataIndex: 'value',
      key: 'value',
      ellipsis: true,
      render: (value: string) => (
        <Space>
          <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>{value}</Text>
          <Tooltip title="Copy value">
            <CopyOutlined
              style={{ color: 'var(--ant-color-text-tertiary)', cursor: 'pointer', fontSize: 12 }}
              onClick={() => handleCopy(value, 'Value')}
            />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: 'TTL',
      dataIndex: 'ttl',
      key: 'ttl',
      width: 80,
      align: 'right' as const,
      sorter: (a: DNSRecord, b: DNSRecord) => a.ttl - b.ttl,
      render: (ttl: number) => (
        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>{ttl}</Text>
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      align: 'right' as const,
      render: (p: number | undefined) => p !== undefined ? p : '---',
    },
    {
      title: 'Port',
      dataIndex: 'port',
      key: 'port',
      width: 80,
      align: 'right' as const,
      render: (p: number | undefined) => p !== undefined ? p : '---',
    },
    {
      title: 'Dynamic',
      dataIndex: 'dynamic',
      key: 'dynamic',
      width: 80,
      render: (dynamic: boolean) => dynamic ? <Tag color="cyan">dyn</Tag> : <Tag>static</Tag>,
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      align: 'center' as const,
      render: (_: unknown, record: DNSRecord) => (
        <Dropdown menu={{ items: getRowActions(record) }} trigger={['click']}>
          <Button type="text" size="small" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
  ];

  const diagColumns: ColumnsType<DiagnosticCheck> = [
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 70,
      align: 'center' as const,
      render: (status: string) => diagStatusIcon[status] || status,
    },
    {
      title: 'Check',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: 'Details',
      dataIndex: 'message',
      key: 'message',
    },
  ];

  const warningZones = zones.filter((z) => z.status !== 'healthy');

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          {selectedZone && activeTab === 'records' && (
            <Button
              type="text"
              icon={<ArrowLeftOutlined />}
              onClick={() => { setSelectedZone(null); setActiveTab('zones'); }}
            />
          )}
          <GlobalOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>
            {selectedZone && activeTab === 'records' ? selectedZone : 'DNS Management'}
          </Title>
          {selectedZone && activeTab === 'records' && (
            <Tag color={zones.find((z) => z.name === selectedZone)?.backend === 'samba' ? 'green' : 'orange'}>
              {zones.find((z) => z.name === selectedZone)?.backend}
            </Tag>
          )}
        </Space>
        <Space>
          <Button icon={<ReloadOutlined />}>Refresh</Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              if (activeTab === 'records' && selectedZone) {
                setEditRecord(null);
                setDrawerOpen(true);
              }
            }}
          >
            {activeTab === 'records' ? 'Add Record' : 'New Zone'}
          </Button>
        </Space>
      </div>

      {/* Alerts for unhealthy zones */}
      {warningZones.length > 0 && activeTab === 'zones' && (
        <Alert
          type="warning"
          showIcon
          message={`${warningZones.length} zone${warningZones.length > 1 ? 's' : ''} need attention`}
          description={warningZones.map((z) => `${z.name} (${z.status})`).join(', ')}
          style={{ marginBottom: 16 }}
          closable
        />
      )}

      <Tabs
        activeKey={activeTab}
        onChange={(key) => {
          setActiveTab(key);
          if (key === 'zones') setSelectedZone(null);
        }}
        items={[
          {
            key: 'zones',
            label: `Zones (${zones.length})`,
            children: (
              <div>
                {/* Zone stats */}
                <Row gutter={16} style={{ marginBottom: 16 }}>
                  <Col span={6}>
                    <Card size="small">
                      <Statistic title="Total Zones" value={zones.length} />
                    </Card>
                  </Col>
                  <Col span={6}>
                    <Card size="small">
                      <Statistic title="Total Records" value={zoneStats.totalRecords} />
                    </Card>
                  </Col>
                  <Col span={6}>
                    <Card size="small">
                      <Statistic
                        title="Samba DNS"
                        value={zoneStats.sambaZones}
                        suffix={<Text type="secondary" style={{ fontSize: 14 }}>zones</Text>}
                      />
                    </Card>
                  </Col>
                  <Col span={6}>
                    <Card size="small">
                      <Statistic
                        title="BIND9 DLZ"
                        value={zoneStats.bindZones}
                        suffix={<Text type="secondary" style={{ fontSize: 14 }}>zones</Text>}
                      />
                    </Card>
                  </Col>
                </Row>

                <Table
                  columns={zoneColumns}
                  dataSource={zones}
                  rowKey="name"
                  loading={loading}
                  pagination={false}
                  size="middle"
                  onRow={(record) => ({
                    onClick: () => { setSelectedZone(record.name); setActiveTab('records'); },
                    style: { cursor: 'pointer' },
                  })}
                />
              </div>
            ),
          },
          {
            key: 'records',
            label: selectedZone ? `Records — ${selectedZone}` : 'Records',
            disabled: !selectedZone,
            children: (
              <div>
                {/* Zone detail header */}
                {currentZone && (
                  <Card
                    size="small"
                    style={{ marginBottom: 12 }}
                    styles={{ body: { padding: '8px 16px' } }}
                  >
                    <Space size={24} wrap>
                      <Space size={6}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Type:</Text>
                        <Tag color={currentZone.type === 'forward' ? 'blue' : 'purple'}>
                          {currentZone.type}
                        </Tag>
                      </Space>
                      <Space size={6}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Backend:</Text>
                        <Tag color={currentZone.backend === 'samba' ? 'green' : 'orange'}>
                          {currentZone.backend}
                        </Tag>
                      </Space>
                      <Space size={6}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Records:</Text>
                        <Text strong>{records.length}</Text>
                      </Space>
                      <Space size={6}>
                        <Text type="secondary" style={{ fontSize: 12 }}>SOA Serial:</Text>
                        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                          {currentZone.soaSerial}
                        </Text>
                      </Space>
                      <Space size={6}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Dynamic:</Text>
                        {currentZone.dynamic
                          ? <Tag color="cyan">Enabled</Tag>
                          : <Tag>Disabled</Tag>
                        }
                      </Space>
                      <Space size={6}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Status:</Text>
                        <Badge status={statusBadge[currentZone.status] || 'default'} text={currentZone.status} />
                      </Space>
                    </Space>
                  </Card>
                )}

                {/* Zone properties (aging/scavenging) */}
                {selectedZone && <ZonePropertiesPanel zoneName={selectedZone} />}

                {/* Record type filter tabs */}
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <Space size={4} wrap>
                    {recordTypeTabs.map((tab) => {
                      const count = tab.key === 'all'
                        ? records.length
                        : tab.key === 'A'
                          ? records.filter((r) => r.type === 'A' || r.type === 'AAAA').length
                          : records.filter((r) => r.type === tab.key).length;
                      return (
                        <Button
                          key={tab.key}
                          type={recordTypeFilter === tab.key ? 'primary' : 'text'}
                          size="small"
                          onClick={() => setRecordTypeFilter(tab.key)}
                        >
                          {tab.label} {count > 0 && <Badge count={count} size="small" style={{ marginLeft: 4 }} />}
                        </Button>
                      );
                    })}
                  </Space>
                  <Input
                    placeholder="Filter records..."
                    prefix={<SearchOutlined />}
                    value={recordSearch}
                    onChange={(e) => setRecordSearch(e.target.value)}
                    style={{ width: 220 }}
                    size="small"
                    allowClear
                  />
                </div>

                <Table
                  columns={recordColumns}
                  dataSource={filteredRecords}
                  rowKey={(r) => `${r.name}-${r.type}-${r.value}`}
                  pagination={false}
                  size="middle"
                  rowSelection={{}}
                />

                {/* CLI equivalent */}
                {selectedZone && (
                  <Card size="small" style={{ marginTop: 16 }}>
                    <Space>
                      <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent:</Text>
                      <Text
                        copyable
                        style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
                      >
                        samba-tool dns query dc1.example.com {selectedZone} @ ALL
                      </Text>
                    </Space>
                  </Card>
                )}
              </div>
            ),
          },
          {
            key: 'diagnostics',
            label: (
              <Space>
                Diagnostics
                {diagnostics.filter((d) => d.status === 'warning' || d.status === 'fail').length > 0 && (
                  <Badge
                    count={diagnostics.filter((d) => d.status === 'warning' || d.status === 'fail').length}
                    size="small"
                    color="orange"
                  />
                )}
              </Space>
            ),
            children: (
              <div>
                <Card
                  title="DNS Health Checks"
                  extra={<Button icon={<ReloadOutlined />} size="small">Run Checks</Button>}
                >
                  <Table
                    columns={diagColumns}
                    dataSource={diagnostics}
                    rowKey="name"
                    pagination={false}
                    size="middle"
                  />
                </Card>

                {/* Summary */}
                {diagnostics.length > 0 && (
                  <Row gutter={16} style={{ marginTop: 16 }}>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Passed"
                          value={diagnostics.filter((d) => d.status === 'pass').length}
                          valueStyle={{ color: '#52c41a' }}
                          prefix={<CheckCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Warnings"
                          value={diagnostics.filter((d) => d.status === 'warning').length}
                          valueStyle={{ color: '#faad14' }}
                          prefix={<WarningOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Failed"
                          value={diagnostics.filter((d) => d.status === 'fail').length}
                          valueStyle={{ color: '#ff4d4f' }}
                          prefix={<ExclamationCircleOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="Info"
                          value={diagnostics.filter((d) => d.status === 'info').length}
                          valueStyle={{ color: '#1677ff' }}
                          prefix={<InfoCircleOutlined />}
                        />
                      </Card>
                    </Col>
                  </Row>
                )}

                {/* CLI equivalent */}
                <Card size="small" style={{ marginTop: 16 }}>
                  <Descriptions column={1} size="small" title="Equivalent CLI Commands">
                    <Descriptions.Item label="Check SRV records">
                      <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                        samba-tool dns query dc1.example.com example.com _ldap._tcp SRV
                      </Text>
                    </Descriptions.Item>
                    <Descriptions.Item label="Zone info">
                      <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                        samba-tool dns zonelist dc1.example.com
                      </Text>
                    </Descriptions.Item>
                    <Descriptions.Item label="SOA check">
                      <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                        dig @dc1.example.com example.com SOA +short
                      </Text>
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              </div>
            ),
          },
          {
            key: 'serverinfo',
            label: 'Server Info',
            children: <ServerInfoTab />,
          },
          {
            key: 'query',
            label: 'Query Tool',
            children: <QueryToolTab />,
          },
          {
            key: 'srv-validator',
            label: 'SRV Validator',
            children: <SRVValidatorTab />,
          },
          {
            key: 'consistency',
            label: 'Consistency',
            children: <ConsistencyTab />,
          },
        ]}
      />

      {/* Create / Edit Record Drawer */}
      {selectedZone && (
        <CreateRecordDrawer
          open={drawerOpen}
          onClose={() => { setDrawerOpen(false); setEditRecord(null); }}
          onSuccess={() => {
            setDrawerOpen(false);
            setEditRecord(null);
            // Re-fetch records for this zone
            api.get<{ records: DNSRecord[] }>(`/dns/zones/${encodeURIComponent(selectedZone)}/records`)
              .then((data) => setRecords(data.records));
          }}
          zoneName={selectedZone}
          editRecord={editRecord}
        />
      )}

      {/* Delete Confirmation Modal */}
      <Modal
        title="Delete DNS Record"
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(null)}
        onOk={handleDeleteConfirm}
        okText="Delete"
        okButtonProps={{ danger: true }}
      >
        {deleteTarget && (
          <Space direction="vertical" size={8}>
            <Text>
              Are you sure you want to delete this record? This action cannot be undone.
            </Text>
            <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
              <Descriptions column={1} size="small">
                <Descriptions.Item label="Name">
                  <Text code>{deleteTarget.name}</Text>
                </Descriptions.Item>
                <Descriptions.Item label="Type">
                  <Tag>{deleteTarget.type}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Value">
                  <Text code style={{ fontSize: 12 }}>{deleteTarget.value}</Text>
                </Descriptions.Item>
              </Descriptions>
            </Card>
            <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
              <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
              <Text
                copyable
                style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
              >
                samba-tool dns delete dc1.example.com {selectedZone} {deleteTarget.name} {deleteTarget.type} {deleteTarget.value}
              </Text>
            </Card>
          </Space>
        )}
      </Modal>
    </div>
  );
}
