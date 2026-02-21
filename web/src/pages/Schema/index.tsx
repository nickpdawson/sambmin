import { useState, useEffect } from 'react';
import {
  Typography, Table, Card, Space, Input, Alert, Tabs, Tag, Descriptions,
  Spin, Badge, Button, Row, Col, Drawer,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DatabaseOutlined, SearchOutlined, ReloadOutlined,
  TagOutlined, FileTextOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

interface SchemaClass {
  cn: string;
  lDAPDisplayName: string;
  description: string;
  category: string;
  subClassOf: string;
  systemOnly: boolean;
  defaultObjectCategory: string;
}

interface SchemaAttribute {
  cn: string;
  lDAPDisplayName: string;
  description: string;
  syntax: string;
  syntaxOID: string;
  singleValued: boolean;
  systemOnly: boolean;
  indexed: boolean;
}

const categoryColors: Record<string, string> = {
  'Structural': 'blue',
  'Abstract': 'purple',
  'Auxiliary': 'green',
  '88 Class': 'orange',
};

const syntaxColors: Record<string, string> = {
  'DN': 'blue',
  'Unicode String': 'green',
  'Integer': 'orange',
  'Boolean': 'red',
  'Large Integer': 'volcano',
  'Generalized Time': 'cyan',
  'Octet String': 'purple',
  'SID': 'magenta',
  'NT Security Descriptor': 'gold',
};

export default function Schema() {
  const [classes, setClasses] = useState<SchemaClass[]>([]);
  const [classesLoading, setClassesLoading] = useState(false);
  const [classesError, setClassesError] = useState<string | null>(null);
  const [classSearch, setClassSearch] = useState('');

  const [attributes, setAttributes] = useState<SchemaAttribute[]>([]);
  const [attrsLoading, setAttrsLoading] = useState(false);
  const [attrsError, setAttrsError] = useState<string | null>(null);
  const [attrSearch, setAttrSearch] = useState('');

  const [selectedClass, setSelectedClass] = useState<SchemaClass | null>(null);
  const [selectedAttr, setSelectedAttr] = useState<SchemaAttribute | null>(null);

  const [activeTab, setActiveTab] = useState('classes');

  useEffect(() => {
    fetchClasses();
  }, []);

  const fetchClasses = () => {
    setClassesLoading(true);
    setClassesError(null);
    api.get<{ classes: SchemaClass[]; total: number }>('/schema/classes')
      .then((data) => setClasses(data.classes || []))
      .catch((err) => setClassesError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setClassesLoading(false));
  };

  const fetchAttributes = () => {
    setAttrsLoading(true);
    setAttrsError(null);
    api.get<{ attributes: SchemaAttribute[]; total: number }>('/schema/attributes')
      .then((data) => setAttributes(data.attributes || []))
      .catch((err) => setAttrsError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setAttrsLoading(false));
  };

  const filteredClasses = classSearch
    ? classes.filter((c) =>
        c.lDAPDisplayName.toLowerCase().includes(classSearch.toLowerCase()) ||
        c.cn.toLowerCase().includes(classSearch.toLowerCase()) ||
        c.description?.toLowerCase().includes(classSearch.toLowerCase()) ||
        c.category?.toLowerCase().includes(classSearch.toLowerCase())
      )
    : classes;

  const filteredAttrs = attrSearch
    ? attributes.filter((a) =>
        a.lDAPDisplayName.toLowerCase().includes(attrSearch.toLowerCase()) ||
        a.cn.toLowerCase().includes(attrSearch.toLowerCase()) ||
        a.description?.toLowerCase().includes(attrSearch.toLowerCase()) ||
        a.syntax?.toLowerCase().includes(attrSearch.toLowerCase())
      )
    : attributes;

  const classColumns: ColumnsType<SchemaClass> = [
    {
      title: 'LDAP Display Name',
      dataIndex: 'lDAPDisplayName',
      key: 'lDAPDisplayName',
      sorter: (a, b) => a.lDAPDisplayName.localeCompare(b.lDAPDisplayName),
      defaultSortOrder: 'ascend',
      render: (val: string, record) => (
        <Button type="link" style={{ ...mono, padding: 0 }} onClick={() => setSelectedClass(record)}>
          {val}
        </Button>
      ),
    },
    {
      title: 'CN',
      dataIndex: 'cn',
      key: 'cn',
      ellipsis: true,
      render: (val: string) => <Text style={{ ...mono, fontSize: 12 }}>{val}</Text>,
    },
    {
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      filters: [
        { text: 'Structural', value: 'Structural' },
        { text: 'Abstract', value: 'Abstract' },
        { text: 'Auxiliary', value: 'Auxiliary' },
        { text: '88 Class', value: '88 Class' },
      ],
      onFilter: (value, record) => record.category === value,
      render: (val: string) => <Tag color={categoryColors[val] || 'default'}>{val}</Tag>,
    },
    {
      title: 'Subclass Of',
      dataIndex: 'subClassOf',
      key: 'subClassOf',
      width: 150,
      render: (val: string) => <Text style={{ ...mono, fontSize: 12 }}>{val}</Text>,
    },
    {
      title: 'System',
      dataIndex: 'systemOnly',
      key: 'systemOnly',
      width: 80,
      filters: [
        { text: 'System', value: true },
        { text: 'Custom', value: false },
      ],
      onFilter: (value, record) => record.systemOnly === value,
      render: (val: boolean) => val ? <Tag color="orange">System</Tag> : <Tag>Custom</Tag>,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 300,
    },
  ];

  const attrColumns: ColumnsType<SchemaAttribute> = [
    {
      title: 'LDAP Display Name',
      dataIndex: 'lDAPDisplayName',
      key: 'lDAPDisplayName',
      sorter: (a, b) => a.lDAPDisplayName.localeCompare(b.lDAPDisplayName),
      defaultSortOrder: 'ascend',
      render: (val: string, record) => (
        <Button type="link" style={{ ...mono, padding: 0 }} onClick={() => setSelectedAttr(record)}>
          {val}
        </Button>
      ),
    },
    {
      title: 'Syntax',
      dataIndex: 'syntax',
      key: 'syntax',
      width: 180,
      filters: [
        { text: 'Unicode String', value: 'Unicode String' },
        { text: 'DN', value: 'DN' },
        { text: 'Integer', value: 'Integer' },
        { text: 'Boolean', value: 'Boolean' },
        { text: 'Large Integer', value: 'Large Integer' },
        { text: 'Generalized Time', value: 'Generalized Time' },
        { text: 'Octet String', value: 'Octet String' },
        { text: 'SID', value: 'SID' },
      ],
      onFilter: (value, record) => record.syntax === value,
      render: (val: string) => <Tag color={syntaxColors[val] || 'default'}>{val}</Tag>,
    },
    {
      title: 'Valued',
      dataIndex: 'singleValued',
      key: 'singleValued',
      width: 100,
      filters: [
        { text: 'Single', value: true },
        { text: 'Multi', value: false },
      ],
      onFilter: (value, record) => record.singleValued === value,
      render: (val: boolean) => val ? 'Single' : <Text type="secondary">Multi</Text>,
    },
    {
      title: 'Indexed',
      dataIndex: 'indexed',
      key: 'indexed',
      width: 80,
      filters: [
        { text: 'Yes', value: true },
        { text: 'No', value: false },
      ],
      onFilter: (value, record) => record.indexed === value,
      render: (val: boolean) => val ? <Badge status="success" text="Yes" /> : <Badge status="default" text="No" />,
    },
    {
      title: 'System',
      dataIndex: 'systemOnly',
      key: 'systemOnly',
      width: 80,
      render: (val: boolean) => val ? <Tag color="orange">System</Tag> : <Tag>Custom</Tag>,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 300,
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <DatabaseOutlined style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
          <Title level={4} style={{ margin: 0 }}>Schema Browser</Title>
        </Space>
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={(key) => {
          setActiveTab(key);
          if (key === 'attributes' && attributes.length === 0) {
            fetchAttributes();
          }
        }}
        items={[
          {
            key: 'classes',
            label: (
              <span>
                <TagOutlined /> Classes
                {classes.length > 0 && <Badge count={classes.length} style={{ marginLeft: 8, backgroundColor: '#52c41a' }} />}
              </span>
            ),
            children: (
              <Space direction="vertical" size={16} style={{ width: '100%' }}>
                <Card size="small">
                  <Row justify="space-between" align="middle">
                    <Col>
                      <Text type="secondary">
                        AD schema class definitions (objectClass=classSchema)
                      </Text>
                    </Col>
                    <Col>
                      <Space>
                        <Input
                          placeholder="Filter classes..."
                          prefix={<SearchOutlined />}
                          value={classSearch}
                          onChange={(e) => setClassSearch(e.target.value)}
                          style={{ width: 260 }}
                          allowClear
                        />
                        <Button
                          icon={<ReloadOutlined />}
                          onClick={fetchClasses}
                          loading={classesLoading}
                        >
                          Refresh
                        </Button>
                      </Space>
                    </Col>
                  </Row>
                </Card>

                {classesLoading && classes.length === 0 && <Spin tip="Loading schema classes..." />}
                {classesError && (
                  <Alert
                    type="error"
                    message="Failed to load classes"
                    description={classesError}
                    action={<Button size="small" onClick={fetchClasses}>Retry</Button>}
                  />
                )}

                {classes.length > 0 && (
                  <Table
                    columns={classColumns}
                    dataSource={filteredClasses}
                    rowKey="cn"
                    size="small"
                    pagination={{
                      pageSize: 50,
                      showSizeChanger: true,
                      pageSizeOptions: ['25', '50', '100', '200'],
                      showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} classes`,
                    }}
                    scroll={{ x: 900 }}
                  />
                )}
              </Space>
            ),
          },
          {
            key: 'attributes',
            label: (
              <span>
                <FileTextOutlined /> Attributes
                {attributes.length > 0 && <Badge count={attributes.length} style={{ marginLeft: 8, backgroundColor: '#1677ff' }} />}
              </span>
            ),
            children: (
              <Space direction="vertical" size={16} style={{ width: '100%' }}>
                <Card size="small">
                  <Row justify="space-between" align="middle">
                    <Col>
                      <Text type="secondary">
                        AD schema attribute definitions (objectClass=attributeSchema)
                      </Text>
                    </Col>
                    <Col>
                      <Space>
                        <Input
                          placeholder="Filter attributes..."
                          prefix={<SearchOutlined />}
                          value={attrSearch}
                          onChange={(e) => setAttrSearch(e.target.value)}
                          style={{ width: 260 }}
                          allowClear
                        />
                        <Button
                          icon={<ReloadOutlined />}
                          onClick={fetchAttributes}
                          loading={attrsLoading}
                        >
                          Refresh
                        </Button>
                      </Space>
                    </Col>
                  </Row>
                </Card>

                {attrsLoading && attributes.length === 0 && <Spin tip="Loading schema attributes..." />}
                {attrsError && (
                  <Alert
                    type="error"
                    message="Failed to load attributes"
                    description={attrsError}
                    action={<Button size="small" onClick={fetchAttributes}>Retry</Button>}
                  />
                )}

                {attributes.length > 0 && (
                  <Table
                    columns={attrColumns}
                    dataSource={filteredAttrs}
                    rowKey="cn"
                    size="small"
                    pagination={{
                      pageSize: 50,
                      showSizeChanger: true,
                      pageSizeOptions: ['25', '50', '100', '200'],
                      showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} attributes`,
                    }}
                    scroll={{ x: 900 }}
                  />
                )}
              </Space>
            ),
          },
        ]}
      />

      {/* Class detail drawer */}
      <Drawer
        title={selectedClass?.lDAPDisplayName || 'Class Details'}
        open={!!selectedClass}
        onClose={() => setSelectedClass(null)}
        width={520}
      >
        {selectedClass && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered size="small" column={1}>
              <Descriptions.Item label="LDAP Display Name">
                <Text strong style={mono}>{selectedClass.lDAPDisplayName}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="CN">
                <Text style={mono}>{selectedClass.cn}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Category">
                <Tag color={categoryColors[selectedClass.category] || 'default'}>
                  {selectedClass.category}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Subclass Of">
                <Text style={mono}>{selectedClass.subClassOf}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="System Only">
                {selectedClass.systemOnly ? (
                  <Tag color="orange">Yes (system-defined)</Tag>
                ) : (
                  <Tag color="green">No (custom)</Tag>
                )}
              </Descriptions.Item>
              {selectedClass.defaultObjectCategory && (
                <Descriptions.Item label="Default Object Category">
                  <Text style={{ ...mono, fontSize: 12, wordBreak: 'break-all' }}>
                    {selectedClass.defaultObjectCategory}
                  </Text>
                </Descriptions.Item>
              )}
              <Descriptions.Item label="Description">
                {selectedClass.description || <Text type="secondary">No description</Text>}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Drawer>

      {/* Attribute detail drawer */}
      <Drawer
        title={selectedAttr?.lDAPDisplayName || 'Attribute Details'}
        open={!!selectedAttr}
        onClose={() => setSelectedAttr(null)}
        width={520}
      >
        {selectedAttr && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions bordered size="small" column={1}>
              <Descriptions.Item label="LDAP Display Name">
                <Text strong style={mono}>{selectedAttr.lDAPDisplayName}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="CN">
                <Text style={mono}>{selectedAttr.cn}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Syntax">
                <Space>
                  <Tag color={syntaxColors[selectedAttr.syntax] || 'default'}>
                    {selectedAttr.syntax}
                  </Tag>
                  <Text type="secondary" style={{ ...mono, fontSize: 11 }}>
                    ({selectedAttr.syntaxOID})
                  </Text>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="Single-Valued">
                {selectedAttr.singleValued ? (
                  <Tag>Single value</Tag>
                ) : (
                  <Tag color="blue">Multi-valued</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="Indexed">
                {selectedAttr.indexed ? (
                  <Badge status="success" text="Yes (searchable)" />
                ) : (
                  <Badge status="default" text="No" />
                )}
              </Descriptions.Item>
              <Descriptions.Item label="System Only">
                {selectedAttr.systemOnly ? (
                  <Tag color="orange">Yes (system-defined)</Tag>
                ) : (
                  <Tag color="green">No (custom)</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="Description">
                {selectedAttr.description || <Text type="secondary">No description</Text>}
              </Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Drawer>
    </div>
  );
}
