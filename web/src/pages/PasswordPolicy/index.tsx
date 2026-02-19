import { useEffect, useState, useCallback } from 'react';
import {
  Button, Card, Space, Typography, Descriptions, Switch, InputNumber, Input,
  notification, Tabs, Table, Modal, Form, Tag, Popconfirm, Alert, Progress, Row, Col,
} from 'antd';
import {
  SafetyCertificateOutlined, PlusOutlined, DeleteOutlined, EditOutlined,
  CheckCircleOutlined, CloseCircleOutlined, UserOutlined, TeamOutlined,
  ReloadOutlined, LockOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Title, Text } = Typography;

interface PasswordPolicy {
  minLength: number;
  maxAge: string;
  minAge: string;
  historyLength: number;
  complexity: boolean;
  reversibleEncryption: boolean;
  lockoutThreshold: number;
  lockoutDuration: string;
  lockoutWindow: string;
  storePlaintext: boolean;
}

interface PSO {
  name: string;
  dn: string;
  precedence: number;
  minLength: number;
  maxAge: string;
  minAge: string;
  historyLength: number;
  complexity: boolean;
  reversibleEncryption: boolean;
  lockoutThreshold: number;
  lockoutDuration: string;
  lockoutWindow: string;
  appliesTo: string[];
}

interface PasswordTestResult {
  valid: boolean;
  errors: string[];
  policy: string;
}

export default function PasswordPolicyPage() {
  const [policy, setPolicy] = useState<PasswordPolicy | null>(null);
  const [psos, setPsos] = useState<PSO[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [editForm] = Form.useForm();
  const [psoModalOpen, setPsoModalOpen] = useState(false);
  const [psoForm] = Form.useForm();
  const [applyModalOpen, setApplyModalOpen] = useState(false);
  const [applyTarget, setApplyTarget] = useState('');
  const [selectedPSO, setSelectedPSO] = useState<string>('');
  const [applyAction, setApplyAction] = useState<'apply' | 'unapply'>('apply');
  // Password tester
  const [testPassword, setTestPassword] = useState('');
  const [testUsername, setTestUsername] = useState('');
  const [testResult, setTestResult] = useState<PasswordTestResult | null>(null);
  const [testLoading, setTestLoading] = useState(false);

  const loadPolicy = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<PasswordPolicy>('/password-policy');
      setPolicy(data);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to load';
      notification.error({ message: 'Load Failed', description: message });
    } finally {
      setLoading(false);
    }
  }, []);

  const loadPSOs = useCallback(async () => {
    try {
      const data = await api.get<PSO[]>('/password-policy/pso');
      setPsos(data || []);
    } catch {
      // PSO list might fail if none exist
    }
  }, []);

  useEffect(() => {
    loadPolicy();
    loadPSOs();
  }, [loadPolicy, loadPSOs]);

  const handleUpdatePolicy = async (values: Partial<PasswordPolicy>) => {
    try {
      await api.put('/password-policy', values);
      notification.success({ message: 'Password policy updated' });
      setEditing(false);
      loadPolicy();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Update failed';
      notification.error({ message: 'Update Failed', description: message });
    }
  };

  const handleCreatePSO = async (values: Partial<PSO>) => {
    try {
      await api.post('/password-policy/pso', values);
      notification.success({ message: 'PSO created' });
      setPsoModalOpen(false);
      psoForm.resetFields();
      loadPSOs();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Create failed';
      notification.error({ message: 'Create Failed', description: message });
    }
  };

  const handleDeletePSO = async (name: string) => {
    try {
      await api.delete(`/password-policy/pso/${encodeURIComponent(name)}`);
      notification.success({ message: 'PSO deleted' });
      loadPSOs();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Delete failed';
      notification.error({ message: 'Delete Failed', description: message });
    }
  };

  const handleApplyUnapply = async () => {
    if (!selectedPSO || !applyTarget) return;
    try {
      await api.post(`/password-policy/pso/${encodeURIComponent(selectedPSO)}/${applyAction}`, { target: applyTarget });
      notification.success({ message: `PSO ${applyAction === 'apply' ? 'applied' : 'removed'}` });
      setApplyModalOpen(false);
      setApplyTarget('');
      loadPSOs();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed';
      notification.error({ message: 'Operation Failed', description: message });
    }
  };

  const handleTestPassword = async () => {
    setTestLoading(true);
    try {
      const result = await api.post<PasswordTestResult>('/password-policy/test', {
        password: testPassword,
        username: testUsername || undefined,
      });
      setTestResult(result);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Test failed';
      notification.error({ message: 'Test Failed', description: message });
    } finally {
      setTestLoading(false);
    }
  };

  const passwordStrength = (password: string): number => {
    let score = 0;
    if (password.length >= 8) score += 25;
    if (password.length >= 12) score += 10;
    if (/[A-Z]/.test(password)) score += 20;
    if (/[a-z]/.test(password)) score += 15;
    if (/[0-9]/.test(password)) score += 15;
    if (/[^A-Za-z0-9]/.test(password)) score += 15;
    return Math.min(score, 100);
  };

  const psoColumns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    {
      title: 'Precedence',
      dataIndex: 'precedence',
      key: 'precedence',
      sorter: (a: PSO, b: PSO) => a.precedence - b.precedence,
    },
    { title: 'Min Length', dataIndex: 'minLength', key: 'minLength' },
    {
      title: 'Complexity',
      dataIndex: 'complexity',
      key: 'complexity',
      render: (v: boolean) => v ? <Tag color="green">On</Tag> : <Tag>Off</Tag>,
    },
    { title: 'Max Age', dataIndex: 'maxAge', key: 'maxAge' },
    { title: 'Lockout', dataIndex: 'lockoutThreshold', key: 'lockout', render: (v: number) => v || 'None' },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, record: PSO) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<UserOutlined />}
            onClick={() => {
              setSelectedPSO(record.name);
              setApplyAction('apply');
              setApplyModalOpen(true);
            }}
          >
            Apply
          </Button>
          <Button
            type="link"
            size="small"
            icon={<TeamOutlined />}
            onClick={() => {
              setSelectedPSO(record.name);
              setApplyAction('unapply');
              setApplyModalOpen(true);
            }}
          >
            Remove
          </Button>
          <Popconfirm title={`Delete PSO "${record.name}"?`} onConfirm={() => handleDeletePSO(record.name)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const strength = passwordStrength(testPassword);
  const strengthColor = strength >= 75 ? '#52c41a' : strength >= 50 ? '#faad14' : '#ff4d4f';

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          <SafetyCertificateOutlined style={{ marginRight: 8 }} />
          Password Policies
        </Title>
        <Button icon={<ReloadOutlined />} onClick={() => { loadPolicy(); loadPSOs(); }}>Refresh</Button>
      </div>

      <Tabs defaultActiveKey="default" items={[
        {
          key: 'default',
          label: 'Domain Default Policy',
          children: (
            <Card loading={loading}>
              {policy && !editing && (
                <>
                  <Descriptions
                    column={2}
                    bordered
                    size="small"
                    extra={<Button icon={<EditOutlined />} onClick={() => { setEditing(true); editForm.setFieldsValue(policy); }}>Edit</Button>}
                  >
                    <Descriptions.Item label="Minimum Password Length">{policy.minLength} characters</Descriptions.Item>
                    <Descriptions.Item label="Password History">{policy.historyLength} remembered</Descriptions.Item>
                    <Descriptions.Item label="Minimum Password Age">{policy.minAge || 'None'}</Descriptions.Item>
                    <Descriptions.Item label="Maximum Password Age">{policy.maxAge || 'None'}</Descriptions.Item>
                    <Descriptions.Item label="Password Complexity">
                      {policy.complexity ? <Tag color="green">Enabled</Tag> : <Tag color="red">Disabled</Tag>}
                    </Descriptions.Item>
                    <Descriptions.Item label="Reversible Encryption">
                      {policy.reversibleEncryption ? <Tag color="red">Enabled</Tag> : <Tag color="green">Disabled</Tag>}
                    </Descriptions.Item>
                    <Descriptions.Item label="Account Lockout Threshold">
                      {policy.lockoutThreshold ? `${policy.lockoutThreshold} attempts` : 'Never (0)'}
                    </Descriptions.Item>
                    <Descriptions.Item label="Lockout Duration">{policy.lockoutDuration || 'N/A'}</Descriptions.Item>
                    <Descriptions.Item label="Lockout Observation Window">{policy.lockoutWindow || 'N/A'}</Descriptions.Item>
                    <Descriptions.Item label="Store Plaintext">
                      {policy.storePlaintext ? <Tag color="red">Yes</Tag> : <Tag color="green">No</Tag>}
                    </Descriptions.Item>
                  </Descriptions>
                </>
              )}

              {editing && (
                <Form form={editForm} layout="vertical" onFinish={handleUpdatePolicy}>
                  <Row gutter={16}>
                    <Col span={8}>
                      <Form.Item name="minLength" label="Minimum Password Length">
                        <InputNumber min={0} max={128} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item name="historyLength" label="Password History Length">
                        <InputNumber min={0} max={24} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item name="lockoutThreshold" label="Lockout Threshold (0=never)">
                        <InputNumber min={0} style={{ width: '100%' }} />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Row gutter={16}>
                    <Col span={8}>
                      <Form.Item name="minAge" label="Minimum Password Age">
                        <Input placeholder="e.g., 1" />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item name="maxAge" label="Maximum Password Age">
                        <Input placeholder="e.g., 42" />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item name="lockoutDuration" label="Lockout Duration (mins)">
                        <Input placeholder="e.g., 30" />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Row gutter={16}>
                    <Col span={8}>
                      <Form.Item name="complexity" label="Password Complexity" valuePropName="checked">
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item name="storePlaintext" label="Store Plaintext" valuePropName="checked">
                        <Switch />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Space>
                    <Button type="primary" htmlType="submit">Save Changes</Button>
                    <Button onClick={() => setEditing(false)}>Cancel</Button>
                  </Space>
                </Form>
              )}
            </Card>
          ),
        },
        {
          key: 'pso',
          label: 'Fine-Grained Policies (PSO)',
          children: (
            <Card
              extra={
                <Button type="primary" icon={<PlusOutlined />} onClick={() => setPsoModalOpen(true)}>
                  Create PSO
                </Button>
              }
            >
              <Alert
                type="info"
                showIcon
                message="Fine-grained password policies override the domain default for specific users or groups. Lower precedence values take priority."
                style={{ marginBottom: 16 }}
              />
              <Table
                dataSource={psos}
                columns={psoColumns}
                rowKey="name"
                size="small"
                pagination={false}
                locale={{ emptyText: 'No fine-grained password policies defined' }}
              />
            </Card>
          ),
        },
        {
          key: 'tester',
          label: <><LockOutlined /> Password Tester</>,
          children: (
            <Card>
              <Row gutter={24}>
                <Col span={12}>
                  <Title level={5}>Test Password Against Policy</Title>
                  <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                    Check if a password meets the domain or user-specific policy requirements.
                  </Text>
                  <Form layout="vertical">
                    <Form.Item label="Password to test">
                      <Input.Password
                        value={testPassword}
                        onChange={(e) => { setTestPassword(e.target.value); setTestResult(null); }}
                        placeholder="Enter a password to test"
                        size="large"
                      />
                    </Form.Item>
                    {testPassword && (
                      <div style={{ marginBottom: 16 }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>Strength</Text>
                        <Progress
                          percent={strength}
                          strokeColor={strengthColor}
                          showInfo={false}
                          size="small"
                        />
                      </div>
                    )}
                    <Form.Item label="Username (optional — checks user-specific PSO)">
                      <Input
                        value={testUsername}
                        onChange={(e) => setTestUsername(e.target.value)}
                        placeholder="e.g., jdoe"
                      />
                    </Form.Item>
                    <Button
                      type="primary"
                      onClick={handleTestPassword}
                      loading={testLoading}
                      disabled={!testPassword}
                      icon={<CheckCircleOutlined />}
                    >
                      Test Password
                    </Button>
                  </Form>
                </Col>
                <Col span={12}>
                  {testResult && (
                    <div style={{ marginTop: 32 }}>
                      {testResult.valid ? (
                        <Alert
                          type="success"
                          showIcon
                          icon={<CheckCircleOutlined />}
                          message="Password meets policy requirements"
                          description={`Tested against ${testResult.policy} policy`}
                        />
                      ) : (
                        <Alert
                          type="error"
                          showIcon
                          icon={<CloseCircleOutlined />}
                          message="Password does not meet policy requirements"
                          description={
                            <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                              {testResult.errors.map((err, i) => (
                                <li key={i}>{err}</li>
                              ))}
                            </ul>
                          }
                        />
                      )}
                    </div>
                  )}
                </Col>
              </Row>
            </Card>
          ),
        },
      ]} />

      {/* Create PSO Modal */}
      <Modal
        title="Create Fine-Grained Password Policy"
        open={psoModalOpen}
        onCancel={() => setPsoModalOpen(false)}
        onOk={() => psoForm.submit()}
        width={600}
      >
        <Form form={psoForm} layout="vertical" onFinish={handleCreatePSO}>
          <Row gutter={16}>
            <Col span={16}>
              <Form.Item name="name" label="Policy Name" rules={[{ required: true, message: 'Required' }]}>
                <Input placeholder="e.g., StrictAdminPolicy" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="precedence" label="Precedence" rules={[{ required: true, message: 'Required' }]}>
                <InputNumber min={1} placeholder="10" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="minLength" label="Min Length">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="historyLength" label="History Length">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="lockoutThreshold" label="Lockout Threshold">
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="minAge" label="Min Age (days)">
                <Input placeholder="1" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="maxAge" label="Max Age (days)">
                <Input placeholder="42" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="lockoutDuration" label="Lockout Duration (mins)">
                <Input placeholder="30" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="complexity" label="Password Complexity" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* Apply/Unapply PSO Modal */}
      <Modal
        title={`${applyAction === 'apply' ? 'Apply' : 'Remove'} PSO: ${selectedPSO}`}
        open={applyModalOpen}
        onCancel={() => setApplyModalOpen(false)}
        onOk={handleApplyUnapply}
        okText={applyAction === 'apply' ? 'Apply' : 'Remove'}
        okButtonProps={{ danger: applyAction === 'unapply' }}
      >
        <Form layout="vertical">
          <Form.Item
            label={`${applyAction === 'apply' ? 'Apply to' : 'Remove from'} (user or group sAMAccountName)`}
          >
            <Input
              value={applyTarget}
              onChange={(e) => setApplyTarget(e.target.value)}
              placeholder="e.g., jdoe or Domain Admins"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
