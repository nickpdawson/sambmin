import { useState, useMemo } from 'react';
import {
  Drawer, Form, Input, Select, InputNumber, Button, Space, Collapse, Typography,
  notification,
} from 'antd';
import { GlobalOutlined } from '@ant-design/icons';

const { Text } = Typography;

type RecordType = 'A' | 'AAAA' | 'CNAME' | 'MX' | 'SRV' | 'TXT' | 'NS' | 'PTR';

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

interface CreateRecordDrawerProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
  zoneName: string;
  /** When set, drawer opens in edit mode with pre-filled values */
  editRecord?: DNSRecord | null;
}

const recordTypeOptions: { value: RecordType; label: string; description: string }[] = [
  { value: 'A', label: 'A', description: 'IPv4 address' },
  { value: 'AAAA', label: 'AAAA', description: 'IPv6 address' },
  { value: 'CNAME', label: 'CNAME', description: 'Canonical name (alias)' },
  { value: 'MX', label: 'MX', description: 'Mail exchange' },
  { value: 'SRV', label: 'SRV', description: 'Service locator' },
  { value: 'TXT', label: 'TXT', description: 'Text record' },
  { value: 'NS', label: 'NS', description: 'Nameserver' },
  { value: 'PTR', label: 'PTR', description: 'Pointer (reverse lookup)' },
];

const defaultTTL = 3600;

/**
 * Build the samba-tool CLI equivalent for the current form values.
 */
function buildCLI(
  zoneName: string,
  values: Record<string, unknown>,
  isEdit: boolean,
): string {
  const action = isEdit ? 'update' : 'add';
  const name = (values.name as string) || '<name>';
  const type = (values.recordType as string) || '<type>';

  let data = '';
  switch (type) {
    case 'A':
    case 'AAAA':
      data = (values.ipAddress as string) || '<ip>';
      break;
    case 'CNAME':
      data = (values.target as string) || '<target>';
      break;
    case 'MX':
      data = `"${(values.mailServer as string) || '<mailserver>'} ${values.priority ?? 10}"`;
      break;
    case 'SRV':
      data = `"${(values.target as string) || '<target>'} ${values.priority ?? 0} ${values.weight ?? 100} ${values.port ?? 0}"`;
      break;
    case 'TXT':
      data = `"${(values.txtValue as string) || '<value>'}"`;
      break;
    case 'NS':
      data = (values.nameserver as string) || '<nameserver>';
      break;
    case 'PTR':
      data = (values.hostname as string) || '<hostname>';
      break;
    default:
      data = '<data>';
  }

  return `samba-tool dns ${action} dc1.dzsec.net ${zoneName} ${name} ${type} ${data}`;
}

export default function CreateRecordDrawer({
  open,
  onClose,
  onSuccess,
  zoneName,
  editRecord,
}: CreateRecordDrawerProps) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const isEdit = !!editRecord;

  // Watch record type to drive adaptive fields
  const recordType = Form.useWatch('recordType', form) as RecordType | undefined;

  // Build initial values from editRecord when in edit mode
  const initialValues = useMemo(() => {
    if (!editRecord) {
      return { recordType: 'A' as RecordType, ttl: defaultTTL };
    }

    const base: Record<string, unknown> = {
      recordType: editRecord.type as RecordType,
      name: editRecord.name,
      ttl: editRecord.ttl,
    };

    switch (editRecord.type) {
      case 'A':
      case 'AAAA':
        base.ipAddress = editRecord.value;
        break;
      case 'CNAME':
        base.target = editRecord.value;
        break;
      case 'MX':
        base.mailServer = editRecord.value;
        base.priority = editRecord.priority;
        break;
      case 'SRV':
        base.target = editRecord.value;
        base.priority = editRecord.priority;
        base.weight = editRecord.weight;
        base.port = editRecord.port;
        break;
      case 'TXT':
        base.txtValue = editRecord.value;
        break;
      case 'NS':
        base.nameserver = editRecord.value;
        break;
      case 'PTR':
        base.hostname = editRecord.value;
        break;
    }

    return base;
  }, [editRecord]);

  const handleSubmit = async () => {
    try {
      await form.validateFields();
      setLoading(true);
      // TODO: POST /api/dns/zones/:zone/records or PUT for edit
      await new Promise((r) => setTimeout(r, 600)); // simulate
      notification.success({
        message: isEdit ? 'Record updated' : 'Record created',
        description: `${form.getFieldValue('recordType')} record for ${form.getFieldValue('name') || '@'} in ${zoneName}`,
      });
      form.resetFields();
      onSuccess();
    } catch {
      // Validation failed
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    form.resetFields();
    onClose();
  };

  // Build CLI preview from current form values
  const allValues = Form.useWatch([], form);
  const cliPreview = useMemo(
    () => buildCLI(zoneName, allValues || {}, isEdit),
    [zoneName, allValues, isEdit],
  );

  return (
    <Drawer
      title={
        <Space>
          <GlobalOutlined />
          <span>{isEdit ? 'Edit Record' : 'Add Record'}</span>
          {zoneName && (
            <Text type="secondary" style={{ fontWeight: 'normal', fontSize: 13 }}>
              -- {zoneName}
            </Text>
          )}
        </Space>
      }
      placement="right"
      width={520}
      open={open}
      onClose={handleClose}
      destroyOnClose
      extra={
        <Space>
          <Button onClick={handleClose}>Cancel</Button>
          <Button type="primary" loading={loading} onClick={handleSubmit}>
            {isEdit ? 'Update Record' : 'Add Record'}
          </Button>
        </Space>
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={initialValues}
        key={editRecord ? `edit-${editRecord.name}-${editRecord.type}` : 'create'}
      >
        {/* Record Type Selector */}
        <Form.Item
          name="recordType"
          label="Record Type"
          rules={[{ required: true, message: 'Select a record type' }]}
        >
          <Select
            options={recordTypeOptions.map((opt) => ({
              value: opt.value,
              label: (
                <Space>
                  <Text strong>{opt.label}</Text>
                  <Text type="secondary" style={{ fontSize: 12 }}>{opt.description}</Text>
                </Space>
              ),
            }))}
            disabled={isEdit}
          />
        </Form.Item>

        {/* Name field -- shown for all types except PTR */}
        {recordType !== 'PTR' && (
          <Form.Item
            name="name"
            label="Name"
            rules={[{ required: true, message: 'Record name is required' }]}
            help={`Use @ for zone root. Relative to ${zoneName}`}
          >
            <Input
              placeholder="@ or subdomain"
              addonAfter={`.${zoneName}`}
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>
        )}

        {/* Type-adaptive fields */}
        {(recordType === 'A' || recordType === 'AAAA') && (
          <Form.Item
            name="ipAddress"
            label={recordType === 'A' ? 'IPv4 Address' : 'IPv6 Address'}
            rules={[
              { required: true, message: 'IP address is required' },
              recordType === 'A'
                ? {
                    pattern: /^(\d{1,3}\.){3}\d{1,3}$/,
                    message: 'Enter a valid IPv4 address (e.g. 10.10.1.10)',
                  }
                : {
                    pattern: /^[0-9a-fA-F:]+$/,
                    message: 'Enter a valid IPv6 address',
                  },
            ]}
          >
            <Input
              placeholder={recordType === 'A' ? '10.10.1.10' : '2001:db8::1'}
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>
        )}

        {recordType === 'CNAME' && (
          <Form.Item
            name="target"
            label="Target"
            rules={[{ required: true, message: 'Target hostname is required' }]}
          >
            <Input
              placeholder="target.dzsec.net"
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>
        )}

        {recordType === 'MX' && (
          <>
            <Form.Item
              name="mailServer"
              label="Mail Server"
              rules={[{ required: true, message: 'Mail server is required' }]}
            >
              <Input
                placeholder="mail.dzsec.net"
                style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              />
            </Form.Item>
            <Form.Item
              name="priority"
              label="Priority"
              rules={[{ required: true, message: 'Priority is required' }]}
              initialValue={10}
            >
              <InputNumber min={0} max={65535} style={{ width: '100%' }} />
            </Form.Item>
          </>
        )}

        {recordType === 'SRV' && (
          <>
            <Form.Item
              name="target"
              label="Target"
              rules={[{ required: true, message: 'Target hostname is required' }]}
            >
              <Input
                placeholder="dc1.dzsec.net"
                style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              />
            </Form.Item>
            <Space style={{ width: '100%' }} size={12}>
              <Form.Item
                name="priority"
                label="Priority"
                rules={[{ required: true, message: 'Required' }]}
                initialValue={0}
                style={{ flex: 1 }}
              >
                <InputNumber min={0} max={65535} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="weight"
                label="Weight"
                rules={[{ required: true, message: 'Required' }]}
                initialValue={100}
                style={{ flex: 1 }}
              >
                <InputNumber min={0} max={65535} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="port"
                label="Port"
                rules={[{ required: true, message: 'Required' }]}
                style={{ flex: 1 }}
              >
                <InputNumber min={0} max={65535} style={{ width: '100%' }} placeholder="389" />
              </Form.Item>
            </Space>
          </>
        )}

        {recordType === 'TXT' && (
          <Form.Item
            name="txtValue"
            label="Value"
            rules={[{ required: true, message: 'TXT value is required' }]}
          >
            <Input.TextArea
              rows={3}
              placeholder='v=spf1 include:_spf.google.com ~all'
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>
        )}

        {recordType === 'NS' && (
          <Form.Item
            name="nameserver"
            label="Nameserver"
            rules={[{ required: true, message: 'Nameserver is required' }]}
          >
            <Input
              placeholder="ns1.dzsec.net"
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>
        )}

        {recordType === 'PTR' && (
          <>
            <Form.Item
              name="name"
              label="IP Address"
              rules={[{ required: true, message: 'IP address is required' }]}
              help="The reverse lookup IP (last octet or full IP depending on zone)"
            >
              <Input
                placeholder="10"
                style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              />
            </Form.Item>
            <Form.Item
              name="hostname"
              label="Hostname"
              rules={[{ required: true, message: 'Hostname is required' }]}
            >
              <Input
                placeholder="dc1.dzsec.net"
                style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              />
            </Form.Item>
          </>
        )}

        {/* TTL -- always shown */}
        <Form.Item
          name="ttl"
          label="TTL (seconds)"
          rules={[{ required: true, message: 'TTL is required' }]}
        >
          <InputNumber
            min={60}
            max={604800}
            style={{ width: '100%' }}
            addonAfter={
              <Select
                defaultValue="custom"
                style={{ width: 110 }}
                size="small"
                onChange={(val: string) => {
                  if (val !== 'custom') form.setFieldsValue({ ttl: Number(val) });
                }}
                options={[
                  { value: 'custom', label: 'Custom' },
                  { value: '300', label: '5 min' },
                  { value: '900', label: '15 min' },
                  { value: '3600', label: '1 hour' },
                  { value: '86400', label: '1 day' },
                  { value: '604800', label: '1 week' },
                ]}
              />
            }
          />
        </Form.Item>

        {/* CLI Equivalent */}
        <Collapse
          ghost
          defaultActiveKey={['cli']}
          items={[
            {
              key: 'cli',
              label: 'CLI Equivalent',
              children: (
                <pre style={{
                  fontSize: 12,
                  fontFamily: '"JetBrains Mono", monospace',
                  background: 'var(--ant-color-bg-layout)',
                  padding: 12,
                  borderRadius: 6,
                  overflowX: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                }}>
                  {cliPreview}
                </pre>
              ),
            },
          ]}
          style={{ marginTop: 8 }}
        />
      </Form>
    </Drawer>
  );
}
