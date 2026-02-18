import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Button, Input, Space, Tag, Tabs, Tooltip, Dropdown, Badge, Typography,
  Drawer, Descriptions, Divider, notification,
} from 'antd';
import {
  DesktopOutlined, ReloadOutlined, MoreOutlined,
  CheckCircleOutlined, StopOutlined, SearchOutlined, CopyOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { api } from '../../api/client';

const { Text, Title } = Typography;

const mono = { fontFamily: '"JetBrains Mono", monospace' };

interface Computer {
  dn: string;
  name: string;
  samAccountName: string;
  dnsHostName: string;
  operatingSystem: string;
  operatingSystemVersion: string;
  site: string;
  enabled: boolean;
  lastLogon: string;
  whenCreated: string;
}

type TabFilter = 'all' | 'active' | 'disabled';

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

export default function Computers() {
  const actionRef = useRef<ActionType>(null);
  const [computers, setComputers] = useState<Computer[]>([]);
  const [loading, setLoading] = useState(true);
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedComputer, setSelectedComputer] = useState<Computer | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  const loadComputers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ computers: Computer[]; total: number }>('/computers');
      setComputers(data.computers);
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadComputers();
  }, [loadComputers]);

  const filteredComputers = computers.filter((c) => {
    if (tabFilter === 'active' && !c.enabled) return false;
    if (tabFilter === 'disabled' && c.enabled) return false;

    if (search) {
      const s = search.toLowerCase();
      return (
        c.name.toLowerCase().includes(s) ||
        c.dnsHostName.toLowerCase().includes(s) ||
        c.operatingSystem.toLowerCase().includes(s)
      );
    }
    return true;
  });

  const disabledCount = computers.filter((c) => !c.enabled).length;
  const activeCount = computers.length - disabledCount;

  const columns: ProColumns<Computer>[] = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      render: (_, record) => (
        <div>
          <a onClick={() => { setSelectedComputer(record); setDrawerOpen(true); }}>
            <Space size={6}>
              <DesktopOutlined />
              {record.name}
            </Space>
          </a>
          <br />
          <Text type="secondary" style={{ fontSize: 12, ...mono }}>
            {record.samAccountName}
          </Text>
        </div>
      ),
    },
    {
      title: 'DNS Hostname',
      dataIndex: 'dnsHostName',
      key: 'dnsHostName',
      ellipsis: true,
      render: (_, record) => (
        <Text style={{ fontSize: 13, ...mono }}>{record.dnsHostName}</Text>
      ),
    },
    {
      title: 'OS',
      dataIndex: 'operatingSystem',
      key: 'operatingSystem',
      ellipsis: true,
      filters: [...new Set(computers.map((c) => c.operatingSystem).filter(Boolean))].map((os) => ({
        text: os,
        value: os,
      })),
      onFilter: (value, record) => record.operatingSystem === value,
      render: (_, record) => (
        <Tooltip title={record.operatingSystemVersion ? `${record.operatingSystem} ${record.operatingSystemVersion}` : undefined}>
          <span>{record.operatingSystem || <Text type="secondary">Unknown</Text>}</span>
        </Tooltip>
      ),
    },
    {
      title: 'Status',
      key: 'status',
      width: 120,
      render: (_, record) => {
        if (!record.enabled) return <Tag icon={<StopOutlined />} color="default">Disabled</Tag>;
        return <Tag icon={<CheckCircleOutlined />} color="success">Active</Tag>;
      },
    },
    {
      title: 'Last Logon',
      dataIndex: 'lastLogon',
      key: 'lastLogon',
      width: 120,
      responsive: ['lg'],
      sorter: (a, b) => new Date(a.lastLogon).getTime() - new Date(b.lastLogon).getTime(),
      render: (_, record) => (
        <Tooltip title={new Date(record.lastLogon).toLocaleString()}>
          <Text type="secondary" style={{ fontSize: 13 }}>{timeAgo(record.lastLogon)}</Text>
        </Tooltip>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'whenCreated',
      key: 'whenCreated',
      width: 120,
      responsive: ['xl'],
      sorter: (a, b) => new Date(a.whenCreated).getTime() - new Date(b.whenCreated).getTime(),
      render: (_, record) => (
        <Text type="secondary" style={{ fontSize: 13 }}>
          {new Date(record.whenCreated).toLocaleDateString()}
        </Text>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      render: (_, record) => (
        <Dropdown
          menu={{
            items: [
              { key: 'view', label: 'View Details' },
              { type: 'divider' },
              record.enabled
                ? { key: 'disable', label: 'Disable Account', danger: true }
                : { key: 'enable', label: 'Enable Account' },
              { type: 'divider' },
              { key: 'delete', label: 'Delete Computer', danger: true },
            ],
            onClick: ({ key }) => {
              if (key === 'view') {
                setSelectedComputer(record);
                setDrawerOpen(true);
              } else {
                notification.info({ message: `${key} — not yet implemented` });
              }
            },
          }}
          trigger={['click']}
        >
          <Button type="text" icon={<MoreOutlined />} size="small" />
        </Dropdown>
      ),
    },
  ];

  const tabItems = [
    { key: 'all', label: `All Computers (${computers.length})` },
    { key: 'active', label: `Active (${activeCount})` },
    { key: 'disabled', label: <Badge count={disabledCount} size="small" offset={[8, 0]}>Disabled</Badge> },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Tabs
        activeKey={tabFilter}
        onChange={(key) => setTabFilter(key as TabFilter)}
        items={tabItems}
        style={{ marginBottom: -8 }}
      />

      <ProTable<Computer>
        actionRef={actionRef}
        columns={columns}
        dataSource={filteredComputers}
        rowKey="dn"
        loading={loading}
        search={false}
        dateFormatter="string"
        options={false}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `${total} computers`,
        }}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as string[]),
        }}
        toolBarRender={() => [
          <Input
            key="search"
            placeholder="Search computers..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            allowClear
            style={{ width: 260 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={loadComputers}>
            Refresh
          </Button>,
        ]}
        headerTitle={
          selectedRowKeys.length > 0 ? (
            <Space>
              <Text>{selectedRowKeys.length} selected</Text>
              <Button size="small" danger onClick={() => notification.info({ message: 'Bulk delete — not yet implemented' })}>
                Delete
              </Button>
              <Button size="small" type="link" onClick={() => setSelectedRowKeys([])}>Clear</Button>
            </Space>
          ) : undefined
        }
      />

      {/* Detail Drawer */}
      <ComputerDrawer
        computer={selectedComputer}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedComputer(null); }}
      />
    </Space>
  );
}

/* ------------------------------------------------------------------ */
/*  Computer Detail Drawer                                            */
/* ------------------------------------------------------------------ */

interface ComputerDrawerProps {
  computer: Computer | null;
  open: boolean;
  onClose: () => void;
}

function ComputerDrawer({ computer, open, onClose }: ComputerDrawerProps) {
  if (!computer) return null;

  const statusTag = !computer.enabled
    ? <Tag icon={<StopOutlined />} color="default">Disabled</Tag>
    : <Tag icon={<CheckCircleOutlined />} color="success">Active</Tag>;

  return (
    <Drawer
      title={
        <Space>
          <DesktopOutlined />
          <span>{computer.name}</span>
          {statusTag}
        </Space>
      }
      placement="right"
      width={560}
      open={open}
      onClose={onClose}
    >
      {/* Identity */}
      <Title level={5} style={{ marginBottom: 12 }}>Computer Identity</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="Name">{computer.name}</Descriptions.Item>
        <Descriptions.Item label="SAM Account Name">
          <Space>
            <Text style={{ ...mono }}>{computer.samAccountName}</Text>
            <Tooltip title="Copy SAM account name">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(computer.samAccountName, 'SAM Account Name')}
              />
            </Tooltip>
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="Distinguished Name">
          <Space>
            <Text style={{ fontSize: 12, wordBreak: 'break-all', ...mono }}>{computer.dn}</Text>
            <Tooltip title="Copy DN">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(computer.dn, 'DN')}
              />
            </Tooltip>
          </Space>
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Network */}
      <Title level={5} style={{ marginBottom: 12 }}>Network</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="DNS Hostname">
          <Space>
            <Text style={{ ...mono }}>{computer.dnsHostName}</Text>
            <Tooltip title="Copy DNS hostname">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(computer.dnsHostName, 'DNS Hostname')}
              />
            </Tooltip>
          </Space>
        </Descriptions.Item>
        <Descriptions.Item label="Site">
          {computer.site || <Text type="secondary">Not assigned</Text>}
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Operating System */}
      <Title level={5} style={{ marginBottom: 12 }}>Operating System</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="OS">
          {computer.operatingSystem || <Text type="secondary">Unknown</Text>}
        </Descriptions.Item>
        <Descriptions.Item label="Version">
          {computer.operatingSystemVersion || <Text type="secondary">Unknown</Text>}
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Account */}
      <Title level={5} style={{ marginBottom: 12 }}>Account</Title>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="Status">{statusTag}</Descriptions.Item>
        <Descriptions.Item label="Last Logon">
          <Tooltip title={new Date(computer.lastLogon).toLocaleString()}>
            {new Date(computer.lastLogon).toLocaleString()}
          </Tooltip>
        </Descriptions.Item>
        <Descriptions.Item label="Created">
          {new Date(computer.whenCreated).toLocaleDateString()}
        </Descriptions.Item>
      </Descriptions>

      <Divider />

      {/* Actions */}
      <Space direction="vertical" style={{ width: '100%' }}>
        {computer.enabled ? (
          <Button block danger onClick={() => notification.info({ message: 'Disable — not yet implemented' })}>
            Disable Computer
          </Button>
        ) : (
          <Button block type="primary" onClick={() => notification.info({ message: 'Enable — not yet implemented' })}>
            Enable Computer
          </Button>
        )}
        <Button block danger type="primary" onClick={() => notification.info({ message: 'Delete — not yet implemented' })}>
          Delete Computer
        </Button>
      </Space>
    </Drawer>
  );
}
