import { useState, useEffect } from 'react';
import {
  Modal, Form, Input, Select, Button, Space, Table, Tag, Typography,
  Popconfirm, notification, InputNumber,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, ArrowUpOutlined, ApiOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

interface DomainController {
  hostname: string;
  address: string;
  port: number;
  site: string;
  primary: boolean;
  status: string;
}

interface ConnectionData {
  domainControllers: DomainController[];
  baseDN: string;
  realm: string;
  protocol: string;
}

interface ConnectionModalProps {
  open: boolean;
  data: ConnectionData;
  onClose: () => void;
  onSave: (data: ConnectionData) => void;
}

export default function ConnectionModal({ open, data, onClose, onSave }: ConnectionModalProps) {
  const [form] = Form.useForm();
  const [dcs, setDcs] = useState<DomainController[]>([]);
  const [loading, setSaving] = useState(false);
  const [testLoading, setTestLoading] = useState(false);
  const [addingDc, setAddingDc] = useState(false);
  const [dcForm] = Form.useForm();

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        baseDN: data.baseDN,
        realm: data.realm,
        protocol: data.protocol,
      });
      setDcs([...data.domainControllers]);
      setAddingDc(false);
    }
  }, [open, data, form]);

  const handleSetPrimary = (hostname: string) => {
    setDcs((prev) =>
      prev.map((dc) => ({
        ...dc,
        primary: dc.hostname === hostname,
      }))
    );
  };

  const handleRemoveDc = (hostname: string) => {
    setDcs((prev) => {
      const next = prev.filter((dc) => dc.hostname !== hostname);
      // If we removed the primary, make the first one primary
      if (next.length > 0 && !next.some((dc) => dc.primary)) {
        next[0].primary = true;
      }
      return next;
    });
  };

  const handleAddDc = async () => {
    try {
      const values = await dcForm.validateFields();
      const newDc: DomainController = {
        hostname: values.hostname,
        address: values.address,
        port: values.port || 636,
        site: values.site || 'Default-First-Site-Name',
        primary: dcs.length === 0,
        status: 'connected',
      };
      setDcs((prev) => [...prev, newDc]);
      dcForm.resetFields();
      setAddingDc(false);
    } catch {
      // Validation failed
    }
  };

  const handleTestConnection = async () => {
    setTestLoading(true);
    // Simulate test
    await new Promise((r) => setTimeout(r, 1200));
    setTestLoading(false);
    notification.success({
      message: 'Connection Successful',
      description: 'All domain controllers are reachable.',
      icon: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    });
  };

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      if (dcs.length === 0) {
        notification.error({ message: 'At least one domain controller is required.' });
        return;
      }
      setSaving(true);
      // Simulate API call
      await new Promise((r) => setTimeout(r, 600));
      onSave({
        baseDN: values.baseDN,
        realm: values.realm,
        protocol: values.protocol,
        domainControllers: dcs,
      });
      notification.success({ message: 'Connection settings saved.' });
      setSaving(false);
    } catch {
      setSaving(false);
    }
  };

  const dcColumns: ColumnsType<DomainController> = [
    {
      title: 'Hostname',
      dataIndex: 'hostname',
      key: 'hostname',
      render: (hostname: string, record) => (
        <Space>
          <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{hostname}</Text>
          {record.primary && <Tag color="blue">Primary</Tag>}
        </Space>
      ),
    },
    {
      title: 'Address',
      dataIndex: 'address',
      key: 'address',
      render: (addr: string) => (
        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{addr}</Text>
      ),
    },
    {
      title: 'Port',
      dataIndex: 'port',
      key: 'port',
      width: 70,
      render: (port: number) => (
        <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>{port}</Text>
      ),
    },
    {
      title: 'Site',
      dataIndex: 'site',
      key: 'site',
      render: (site: string) => <Tag>{site}</Tag>,
    },
    {
      title: '',
      key: 'actions',
      width: 80,
      render: (_, record) => (
        <Space size={4}>
          {!record.primary && (
            <Button
              type="text"
              size="small"
              icon={<ArrowUpOutlined />}
              onClick={() => handleSetPrimary(record.hostname)}
              title="Set as primary"
            />
          )}
          <Popconfirm
            title="Remove this domain controller?"
            onConfirm={() => handleRemoveDc(record.hostname)}
            okText="Remove"
            okButtonProps={{ danger: true }}
          >
            <Button type="text" size="small" icon={<DeleteOutlined />} danger />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Modal
      title="Edit Connection Settings"
      open={open}
      onCancel={onClose}
      width={720}
      footer={
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Button
            icon={<ApiOutlined />}
            loading={testLoading}
            onClick={handleTestConnection}
          >
            Test Connection
          </Button>
          <Space>
            <Button onClick={onClose}>Cancel</Button>
            <Button type="primary" loading={loading} onClick={handleSave}>
              Save Changes
            </Button>
          </Space>
        </div>
      }
    >
      <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
        <Space style={{ width: '100%', display: 'flex' }} size={12}>
          <Form.Item
            name="baseDN"
            label="Base DN"
            rules={[{ required: true, message: 'Required' }]}
            style={{ flex: 2 }}
          >
            <Input
              placeholder="DC=example,DC=com"
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>

          <Form.Item
            name="realm"
            label="Realm"
            rules={[{ required: true, message: 'Required' }]}
            style={{ flex: 1 }}
          >
            <Input
              placeholder="EXAMPLE.COM"
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>

          <Form.Item
            name="protocol"
            label="Protocol"
            rules={[{ required: true, message: 'Required' }]}
            style={{ flex: 1 }}
          >
            <Select
              options={[
                { value: 'ldaps', label: 'LDAPS (636)' },
                { value: 'ldap', label: 'LDAP (389)' },
                { value: 'ldap+starttls', label: 'LDAP + StartTLS' },
              ]}
            />
          </Form.Item>
        </Space>
      </Form>

      <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong>Domain Controllers</Text>
        <Button
          type="dashed"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => setAddingDc(true)}
        >
          Add DC
        </Button>
      </div>

      <Table
        columns={dcColumns}
        dataSource={dcs}
        rowKey="hostname"
        pagination={false}
        size="small"
      />

      {addingDc && (
        <div style={{
          marginTop: 12,
          padding: 16,
          background: 'var(--ant-color-bg-layout)',
          borderRadius: 8,
        }}>
          <Text strong style={{ display: 'block', marginBottom: 12 }}>Add Domain Controller</Text>
          <Form form={dcForm} layout="vertical">
            <Space style={{ width: '100%', display: 'flex' }} size={12}>
              <Form.Item
                name="hostname"
                label="Hostname"
                rules={[{ required: true, message: 'Required' }]}
                style={{ flex: 2 }}
              >
                <Input
                  placeholder="dc3.example.com"
                  style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
                />
              </Form.Item>
              <Form.Item
                name="address"
                label="IP Address"
                rules={[{ required: true, message: 'Required' }]}
                style={{ flex: 1 }}
              >
                <Input
                  placeholder="10.0.0.3"
                  style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
                />
              </Form.Item>
              <Form.Item name="port" label="Port" style={{ flex: 0, minWidth: 80 }}>
                <InputNumber placeholder="636" min={1} max={65535} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name="site" label="Site" style={{ flex: 1 }}>
                <Input placeholder="Default-First-Site-Name" />
              </Form.Item>
            </Space>
            <Space>
              <Button type="primary" size="small" onClick={handleAddDc}>Add</Button>
              <Button size="small" onClick={() => { dcForm.resetFields(); setAddingDc(false); }}>Cancel</Button>
            </Space>
          </Form>
        </div>
      )}
    </Modal>
  );
}
