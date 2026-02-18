import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Button, Input, Space, Tag, Tabs, Tooltip, Typography, Drawer, Descriptions,
  Divider, List, notification,
} from 'antd';
import {
  PlusOutlined, ReloadOutlined, SearchOutlined, TeamOutlined,
  SafetyCertificateOutlined, MailOutlined, CopyOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { api } from '../../api/client';

const { Text, Title } = Typography;

interface Group {
  dn: string;
  name: string;
  samAccountName: string;
  description: string;
  groupType: string;
  groupScope: string;
  members: string[];
  memberOf: string[];
}

type TabFilter = 'all' | 'security' | 'distribution';

const cnFromDN = (dn: string) => dn.split(',')[0]?.replace(/^CN=/i, '') || dn;

const monoStyle: React.CSSProperties = { fontFamily: '"JetBrains Mono", monospace' };

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

function scopeColor(scope: string): string {
  switch (scope) {
    case 'global': return 'blue';
    case 'domainLocal': return 'orange';
    case 'universal': return 'purple';
    default: return 'default';
  }
}

function scopeLabel(scope: string): string {
  switch (scope) {
    case 'global': return 'Global';
    case 'domainLocal': return 'Domain Local';
    case 'universal': return 'Universal';
    default: return scope;
  }
}

export default function Groups() {
  const actionRef = useRef<ActionType>(null);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [tabFilter, setTabFilter] = useState<TabFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const loadGroups = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ groups: Group[]; total: number }>('/groups');
      setGroups(data.groups);
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadGroups();
  }, [loadGroups]);

  const filteredGroups = groups.filter((g) => {
    if (tabFilter === 'security' && g.groupType !== 'security') return false;
    if (tabFilter === 'distribution' && g.groupType !== 'distribution') return false;

    if (search) {
      const s = search.toLowerCase();
      return (
        g.name.toLowerCase().includes(s) ||
        g.samAccountName.toLowerCase().includes(s) ||
        g.description.toLowerCase().includes(s)
      );
    }
    return true;
  });

  const securityCount = groups.filter((g) => g.groupType === 'security').length;
  const distributionCount = groups.filter((g) => g.groupType === 'distribution').length;

  const columns: ProColumns<Group>[] = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
      render: (_, record) => (
        <div>
          <a onClick={() => { setSelectedGroup(record); setDrawerOpen(true); }}>
            {record.name}
          </a>
          <br />
          <Text type="secondary" style={{ fontSize: 12, ...monoStyle }}>
            {record.samAccountName}
          </Text>
        </div>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'groupType',
      key: 'groupType',
      width: 130,
      render: (_, record) =>
        record.groupType === 'security' ? (
          <Tag icon={<SafetyCertificateOutlined />} color="green">Security</Tag>
        ) : (
          <Tag icon={<MailOutlined />} color="blue">Distribution</Tag>
        ),
    },
    {
      title: 'Scope',
      dataIndex: 'groupScope',
      key: 'groupScope',
      width: 140,
      filters: [
        { text: 'Global', value: 'global' },
        { text: 'Domain Local', value: 'domainLocal' },
        { text: 'Universal', value: 'universal' },
      ],
      onFilter: (value, record) => record.groupScope === value,
      render: (_, record) => (
        <Tag color={scopeColor(record.groupScope)}>
          {scopeLabel(record.groupScope)}
        </Tag>
      ),
    },
    {
      title: 'Members',
      key: 'members',
      width: 100,
      sorter: (a, b) => a.members.length - b.members.length,
      render: (_, record) => (
        <Space size={4}>
          <TeamOutlined />
          <span>{record.members.length}</span>
        </Space>
      ),
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      responsive: ['lg'],
    },
  ];

  const tabItems = [
    { key: 'all', label: `All Groups (${groups.length})` },
    { key: 'security', label: `Security (${securityCount})` },
    { key: 'distribution', label: `Distribution (${distributionCount})` },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Tabs
        activeKey={tabFilter}
        onChange={(key) => setTabFilter(key as TabFilter)}
        items={tabItems}
        style={{ marginBottom: -8 }}
      />

      <ProTable<Group>
        actionRef={actionRef}
        columns={columns}
        dataSource={filteredGroups}
        rowKey="dn"
        loading={loading}
        search={false}
        dateFormatter="string"
        options={false}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `${total} groups`,
        }}
        toolBarRender={() => [
          <Input
            key="search"
            placeholder="Search groups..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            allowClear
            style={{ width: 240 }}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={loadGroups} />,
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => notification.info({ message: 'New Group — not yet implemented' })}
          >
            New Group
          </Button>,
        ]}
      />

      {/* Group Detail Drawer */}
      <Drawer
        title={
          selectedGroup ? (
            <Space>
              <TeamOutlined />
              <span>{selectedGroup.name}</span>
              {selectedGroup.groupType === 'security' ? (
                <Tag icon={<SafetyCertificateOutlined />} color="green">Security</Tag>
              ) : (
                <Tag icon={<MailOutlined />} color="blue">Distribution</Tag>
              )}
              <Tag color={scopeColor(selectedGroup.groupScope)}>
                {scopeLabel(selectedGroup.groupScope)}
              </Tag>
            </Space>
          ) : null
        }
        placement="right"
        width={560}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedGroup(null); }}
      >
        {selectedGroup && (
          <>
            {/* Identity */}
            <Title level={5} style={{ marginBottom: 12 }}>Identity</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Name">{selectedGroup.name}</Descriptions.Item>
              <Descriptions.Item label="sAMAccountName">
                <Space>
                  <Text code style={monoStyle}>{selectedGroup.samAccountName}</Text>
                  <Tooltip title="Copy sAMAccountName">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => copyToClipboard(selectedGroup.samAccountName, 'sAMAccountName')}
                    />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="DN">
                <Space>
                  <Text
                    code
                    style={{ fontSize: 12, wordBreak: 'break-all', ...monoStyle }}
                  >
                    {selectedGroup.dn}
                  </Text>
                  <Tooltip title="Copy DN">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => copyToClipboard(selectedGroup.dn, 'DN')}
                    />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
            </Descriptions>

            <Divider />

            {/* Description */}
            <Title level={5} style={{ marginBottom: 12 }}>Description</Title>
            <Text type={selectedGroup.description ? undefined : 'secondary'}>
              {selectedGroup.description || 'No description'}
            </Text>

            <Divider />

            {/* Type & Scope */}
            <Title level={5} style={{ marginBottom: 12 }}>Type & Scope</Title>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="Type">
                {selectedGroup.groupType === 'security' ? (
                  <Tag icon={<SafetyCertificateOutlined />} color="green">Security</Tag>
                ) : (
                  <Tag icon={<MailOutlined />} color="blue">Distribution</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="Scope">
                <Tag color={scopeColor(selectedGroup.groupScope)}>
                  {scopeLabel(selectedGroup.groupScope)}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Divider />

            {/* Members */}
            <Title level={5} style={{ marginBottom: 12 }}>
              Members ({selectedGroup.members.length})
            </Title>
            {selectedGroup.members.length > 0 ? (
              <List
                size="small"
                bordered
                dataSource={selectedGroup.members}
                renderItem={(memberDN) => (
                  <List.Item>
                    <Space direction="vertical" size={0} style={{ width: '100%' }}>
                      <Text>{cnFromDN(memberDN)}</Text>
                      <Text type="secondary" style={{ fontSize: 11, ...monoStyle }}>
                        {memberDN}
                      </Text>
                    </Space>
                  </List.Item>
                )}
              />
            ) : (
              <Text type="secondary">No members</Text>
            )}

            <Divider />

            {/* Member Of */}
            <Title level={5} style={{ marginBottom: 12 }}>
              Member Of ({selectedGroup.memberOf.length})
            </Title>
            {selectedGroup.memberOf.length > 0 ? (
              <Space size={[4, 8]} wrap>
                {selectedGroup.memberOf.map((parentDN) => (
                  <Tag key={parentDN} style={{ cursor: 'default' }}>
                    {cnFromDN(parentDN)}
                  </Tag>
                ))}
              </Space>
            ) : (
              <Text type="secondary">Not a member of any groups</Text>
            )}
          </>
        )}
      </Drawer>
    </Space>
  );
}
