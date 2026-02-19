import { useState, useCallback } from 'react';
import {
  Button, Input, Select, Space, Card, Tabs, Typography, Tag, Table,
  notification, Form, Row, Col, Tooltip, Popconfirm, List, Empty, Modal,
} from 'antd';
import {
  SearchOutlined, PlusOutlined, DeleteOutlined, SaveOutlined,
  CodeOutlined, FilterOutlined, PlayCircleOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Text, Title } = Typography;
const { TextArea } = Input;

interface SearchFilter {
  attribute: string;
  operator: string;
  value: string;
}

interface SearchRequest {
  baseDn: string;
  scope: string;
  objectType: string;
  filters: SearchFilter[];
  rawFilter: string;
  attributes: string[];
}

interface SearchResultEntry {
  dn: string;
  objectType: string;
  attributes: Record<string, string>;
}

interface SearchResponse {
  results: SearchResultEntry[];
  count: number;
  filter: string;
}

interface SavedQuery {
  id: string;
  name: string;
  description: string;
  request: SearchRequest;
  createdBy: string;
  createdAt: string;
}

const OPERATORS = [
  { value: 'equals', label: 'Equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'startsWith', label: 'Starts With' },
  { value: 'endsWith', label: 'Ends With' },
  { value: 'present', label: 'Has Value' },
  { value: 'notPresent', label: 'No Value' },
  { value: 'greaterThan', label: '>=' },
  { value: 'lessThan', label: '<=' },
  { value: 'bitwiseAnd', label: 'Bitwise AND' },
  { value: 'bitwiseOr', label: 'Bitwise OR' },
];

const COMMON_ATTRIBUTES = [
  'cn', 'sAMAccountName', 'displayName', 'mail', 'description',
  'department', 'title', 'company', 'memberOf', 'userAccountControl',
  'whenCreated', 'whenChanged', 'lastLogon', 'operatingSystem',
  'dNSHostName', 'groupType', 'objectClass', 'distinguishedName',
  'givenName', 'sn', 'telephoneNumber', 'mobile', 'streetAddress',
  'l', 'st', 'postalCode', 'co', 'manager', 'userPrincipalName',
];

const OBJECT_TYPES = [
  { value: 'all', label: 'All Objects' },
  { value: 'user', label: 'Users' },
  { value: 'group', label: 'Groups' },
  { value: 'computer', label: 'Computers' },
  { value: 'contact', label: 'Contacts' },
];

const SCOPES = [
  { value: 'sub', label: 'Subtree (default)' },
  { value: 'one', label: 'One Level' },
  { value: 'base', label: 'Base Object' },
];

const QUERY_TEMPLATES = [
  { name: 'All disabled users', filter: '(&(objectClass=user)(!(objectClass=computer))(userAccountControl:1.2.840.113556.1.4.803:=2))' },
  { name: 'All locked accounts', filter: '(&(objectClass=user)(!(objectClass=computer))(lockoutTime>=1))' },
  { name: 'Users without email', filter: '(&(objectClass=user)(!(objectClass=computer))(!(mail=*)))' },
  { name: 'Domain Admins members', filter: '(&(objectClass=user)(memberOf=CN=Domain Admins,CN=Users,DC=dzsec,DC=net))' },
  { name: 'Computers with old OS', filter: '(&(objectClass=computer)(operatingSystem=Windows Server 2012*))' },
  { name: 'Empty groups', filter: '(&(objectClass=group)(!(member=*)))' },
  { name: 'Recently created objects', filter: '(&(objectClass=*)(whenCreated>=20260101000000.0Z))' },
];

export default function Search() {
  const [mode, setMode] = useState<'visual' | 'raw'>('visual');
  const [filters, setFilters] = useState<SearchFilter[]>([{ attribute: 'cn', operator: 'contains', value: '' }]);
  const [rawFilter, setRawFilter] = useState('');
  const [objectType, setObjectType] = useState('all');
  const [scope, setScope] = useState('sub');
  const [baseDn, setBaseDn] = useState('');
  const [results, setResults] = useState<SearchResultEntry[]>([]);
  const [resultCount, setResultCount] = useState(0);
  const [appliedFilter, setAppliedFilter] = useState('');
  const [loading, setLoading] = useState(false);
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [savedLoading, setSavedLoading] = useState(false);
  const [saveModalOpen, setSaveModalOpen] = useState(false);
  const [saveForm] = Form.useForm();

  const loadSavedQueries = useCallback(async () => {
    setSavedLoading(true);
    try {
      const data = await api.get<SavedQuery[]>('/search/saved');
      setSavedQueries(data || []);
    } catch {
      // Silently handle — saved queries are optional
    } finally {
      setSavedLoading(false);
    }
  }, []);

  const handleSearch = useCallback(async () => {
    setLoading(true);
    try {
      const request: SearchRequest = {
        baseDn: baseDn,
        scope,
        objectType: mode === 'visual' ? objectType : 'all',
        filters: mode === 'visual' ? filters.filter(f => f.attribute && (f.value || f.operator === 'present' || f.operator === 'notPresent')) : [],
        rawFilter: mode === 'raw' ? rawFilter : '',
        attributes: [],
      };
      const data = await api.post<SearchResponse>('/search', request);
      setResults(data.results || []);
      setResultCount(data.count);
      setAppliedFilter(data.filter);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Search failed';
      notification.error({ message: 'Search Failed', description: message });
    } finally {
      setLoading(false);
    }
  }, [mode, filters, rawFilter, objectType, scope, baseDn]);

  const addFilter = () => {
    setFilters([...filters, { attribute: 'cn', operator: 'contains', value: '' }]);
  };

  const removeFilter = (index: number) => {
    setFilters(filters.filter((_, i) => i !== index));
  };

  const updateFilter = (index: number, field: keyof SearchFilter, value: string) => {
    const updated = [...filters];
    updated[index] = { ...updated[index], [field]: value };
    setFilters(updated);
  };

  const handleSave = async (values: { name: string; description: string }) => {
    try {
      const request: SearchRequest = {
        baseDn,
        scope,
        objectType: mode === 'visual' ? objectType : 'all',
        filters: mode === 'visual' ? filters : [],
        rawFilter: mode === 'raw' ? rawFilter : '',
        attributes: [],
      };
      await api.post('/search/saved', { name: values.name, description: values.description, request });
      notification.success({ message: 'Query saved' });
      setSaveModalOpen(false);
      saveForm.resetFields();
      loadSavedQueries();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to save query';
      notification.error({ message: 'Save Failed', description: message });
    }
  };

  const handleDeleteSaved = async (id: string) => {
    try {
      await api.delete(`/search/saved/${id}`);
      notification.success({ message: 'Query deleted' });
      loadSavedQueries();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to delete query';
      notification.error({ message: 'Delete Failed', description: message });
    }
  };

  const handleLoadQuery = (query: SavedQuery) => {
    if (query.request.rawFilter) {
      setMode('raw');
      setRawFilter(query.request.rawFilter);
    } else {
      setMode('visual');
      setFilters(query.request.filters?.length ? query.request.filters : [{ attribute: 'cn', operator: 'contains', value: '' }]);
      setObjectType(query.request.objectType || 'all');
    }
    setScope(query.request.scope || 'sub');
    setBaseDn(query.request.baseDn || '');
  };

  const handleLoadTemplate = (filter: string) => {
    setMode('raw');
    setRawFilter(filter);
  };

  const objectTypeColor = (type: string) => {
    switch (type) {
      case 'user': return 'blue';
      case 'group': return 'green';
      case 'computer': return 'orange';
      case 'contact': return 'purple';
      case 'ou': return 'cyan';
      default: return 'default';
    }
  };

  const resultColumns = [
    {
      title: 'Type',
      dataIndex: 'objectType',
      key: 'objectType',
      width: 100,
      render: (type: string) => <Tag color={objectTypeColor(type)}>{type}</Tag>,
    },
    {
      title: 'DN',
      dataIndex: 'dn',
      key: 'dn',
      ellipsis: true,
      render: (dn: string) => (
        <Text style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: 12 }}>{dn}</Text>
      ),
    },
    {
      title: 'Name',
      key: 'name',
      width: 200,
      render: (_: unknown, record: SearchResultEntry) =>
        record.attributes.displayName || record.attributes.cn || record.attributes.sAMAccountName || '—',
    },
    {
      title: 'Description',
      key: 'description',
      width: 250,
      ellipsis: true,
      render: (_: unknown, record: SearchResultEntry) => record.attributes.description || '—',
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Advanced Search</Title>
        <Space>
          <Button icon={<SaveOutlined />} onClick={() => { setSaveModalOpen(true); loadSavedQueries(); }}>
            Save Query
          </Button>
        </Space>
      </div>

      <Row gutter={16}>
        <Col span={18}>
          <Card size="small">
            <Tabs
              activeKey={mode}
              onChange={(key) => setMode(key as 'visual' | 'raw')}
              items={[
                {
                  key: 'visual',
                  label: <><FilterOutlined /> Visual Builder</>,
                  children: (
                    <div>
                      <Row gutter={8} style={{ marginBottom: 8 }}>
                        <Col span={8}>
                          <Select
                            value={objectType}
                            onChange={setObjectType}
                            options={OBJECT_TYPES}
                            style={{ width: '100%' }}
                            placeholder="Object type"
                          />
                        </Col>
                        <Col span={8}>
                          <Select
                            value={scope}
                            onChange={setScope}
                            options={SCOPES}
                            style={{ width: '100%' }}
                          />
                        </Col>
                        <Col span={8}>
                          <Input
                            placeholder="Base DN (optional — uses default)"
                            value={baseDn}
                            onChange={(e) => setBaseDn(e.target.value)}
                            style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: 12 }}
                          />
                        </Col>
                      </Row>

                      {filters.map((filter, index) => (
                        <Row key={index} gutter={8} style={{ marginBottom: 8 }}>
                          <Col span={8}>
                            <Select
                              value={filter.attribute}
                              onChange={(val) => updateFilter(index, 'attribute', val)}
                              showSearch
                              style={{ width: '100%' }}
                              placeholder="Attribute"
                              options={COMMON_ATTRIBUTES.map(a => ({ value: a, label: a }))}
                            />
                          </Col>
                          <Col span={6}>
                            <Select
                              value={filter.operator}
                              onChange={(val) => updateFilter(index, 'operator', val)}
                              options={OPERATORS}
                              style={{ width: '100%' }}
                            />
                          </Col>
                          <Col span={8}>
                            <Input
                              value={filter.value}
                              onChange={(e) => updateFilter(index, 'value', e.target.value)}
                              placeholder="Value"
                              disabled={filter.operator === 'present' || filter.operator === 'notPresent'}
                              onPressEnter={handleSearch}
                            />
                          </Col>
                          <Col span={2}>
                            {filters.length > 1 && (
                              <Button
                                type="text"
                                danger
                                icon={<DeleteOutlined />}
                                onClick={() => removeFilter(index)}
                              />
                            )}
                          </Col>
                        </Row>
                      ))}

                      <Button type="dashed" onClick={addFilter} icon={<PlusOutlined />} style={{ marginBottom: 8 }}>
                        Add Condition
                      </Button>
                    </div>
                  ),
                },
                {
                  key: 'raw',
                  label: <><CodeOutlined /> Raw LDAP Filter</>,
                  children: (
                    <div>
                      <TextArea
                        value={rawFilter}
                        onChange={(e) => setRawFilter(e.target.value)}
                        placeholder="(&(objectClass=user)(!(objectClass=computer))(department=Engineering))"
                        rows={4}
                        style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: 13 }}
                      />
                      <div style={{ marginTop: 8 }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          Templates:{' '}
                          {QUERY_TEMPLATES.map((t) => (
                            <Tag
                              key={t.name}
                              style={{ cursor: 'pointer', marginBottom: 4 }}
                              onClick={() => handleLoadTemplate(t.filter)}
                            >
                              {t.name}
                            </Tag>
                          ))}
                        </Text>
                      </div>
                    </div>
                  ),
                },
              ]}
            />

            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8 }}>
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                onClick={handleSearch}
                loading={loading}
                size="large"
              >
                Search
              </Button>
            </div>
          </Card>

          {/* Results */}
          <Card
            size="small"
            style={{ marginTop: 16 }}
            title={
              <Space>
                <SearchOutlined />
                <span>Results ({resultCount})</span>
                {appliedFilter && (
                  <Tooltip title={appliedFilter}>
                    <Tag style={{ fontFamily: 'JetBrains Mono, monospace', fontSize: 11, maxWidth: 400, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {appliedFilter}
                    </Tag>
                  </Tooltip>
                )}
              </Space>
            }
          >
            {results.length > 0 ? (
              <Table
                dataSource={results}
                columns={resultColumns}
                rowKey="dn"
                size="small"
                pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (total) => `${total} results` }}
                scroll={{ x: 800 }}
              />
            ) : (
              <Empty description={appliedFilter ? 'No results found' : 'Run a search to see results'} />
            )}
          </Card>
        </Col>

        {/* Saved Queries Sidebar */}
        <Col span={6}>
          <Card size="small" title="Saved Queries" extra={<Button type="text" size="small" onClick={loadSavedQueries}>Refresh</Button>}>
            {savedQueries.length > 0 ? (
              <List
                loading={savedLoading}
                dataSource={savedQueries}
                size="small"
                renderItem={(query) => (
                  <List.Item
                    actions={[
                      <Popconfirm
                        key="delete"
                        title="Delete this saved query?"
                        onConfirm={() => handleDeleteSaved(query.id)}
                      >
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>,
                    ]}
                  >
                    <List.Item.Meta
                      title={
                        <a onClick={() => handleLoadQuery(query)} style={{ cursor: 'pointer' }}>
                          {query.name}
                        </a>
                      }
                      description={query.description || `by ${query.createdBy}`}
                    />
                  </List.Item>
                )}
              />
            ) : (
              <Empty description="No saved queries" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>

          <Card size="small" title="Common Templates" style={{ marginTop: 16 }}>
            <List
              dataSource={QUERY_TEMPLATES}
              size="small"
              renderItem={(template) => (
                <List.Item>
                  <a onClick={() => handleLoadTemplate(template.filter)} style={{ cursor: 'pointer', fontSize: 13 }}>
                    {template.name}
                  </a>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>

      {/* Save Query Modal */}
      <Modal
        title="Save Query"
        open={saveModalOpen}
        onCancel={() => setSaveModalOpen(false)}
        onOk={() => saveForm.submit()}
      >
        <Form form={saveForm} layout="vertical" onFinish={handleSave}>
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}>
            <Input placeholder="My search query" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Optional description" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
