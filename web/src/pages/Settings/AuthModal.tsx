import { useState, useEffect } from 'react';
import {
  Modal, Form, Input, Switch, Slider, Space, Typography, Divider,
  notification, Card,
} from 'antd';
import { KeyOutlined } from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface AuthData {
  kerberos: {
    enabled: boolean;
    implementation: string;
    keytab: string;
    spn: string;
  };
  ldapBind: {
    enabled: boolean;
  };
  sessionTimeout: number;
}

interface AuthModalProps {
  open: boolean;
  data: AuthData;
  onClose: () => void;
  onSave: (data: AuthData, restartRequired?: boolean, restartFields?: string[]) => void;
}

export default function AuthModal({ open, data, onClose, onSave }: AuthModalProps) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [kerberosEnabled, setKerberosEnabled] = useState(data.kerberos.enabled);
  const [ldapEnabled, setLdapEnabled] = useState(data.ldapBind.enabled);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        kerberosEnabled: data.kerberos.enabled,
        keytab: data.kerberos.keytab,
        spn: data.kerberos.spn,
        ldapEnabled: data.ldapBind.enabled,
        sessionTimeout: data.sessionTimeout,
      });
      setKerberosEnabled(data.kerberos.enabled);
      setLdapEnabled(data.ldapBind.enabled);
    }
  }, [open, data, form]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const result = await api.put<{ status: string; restartRequired: boolean; restartFields: string[] }>(
        '/settings/auth',
        {
          kerberos: {
            enabled: values.kerberosEnabled,
            keytab: values.keytab,
          },
          sessionTimeout: values.sessionTimeout,
        },
      );

      const updated: AuthData = {
        kerberos: {
          enabled: values.kerberosEnabled,
          implementation: data.kerberos.implementation,
          keytab: values.keytab,
          spn: values.spn,
        },
        ldapBind: {
          enabled: values.ldapEnabled,
        },
        sessionTimeout: values.sessionTimeout,
      };

      onSave(updated, result.restartRequired, result.restartFields);
      notification.success({ message: 'Authentication settings saved.' });
      if (result.restartRequired) {
        notification.warning({
          message: 'Restart Required',
          description: `Changes to ${result.restartFields.join(', ')} will take effect after server restart.`,
          duration: 8,
        });
      }
      setLoading(false);
    } catch (err: any) {
      notification.error({ message: err?.message || 'Failed to save auth settings.' });
      setLoading(false);
    }
  };

  const sessionTimeoutMarks: Record<number, string> = {
    1: '1h',
    4: '4h',
    8: '8h',
    12: '12h',
    24: '24h',
  };

  return (
    <Modal
      title={
        <Space>
          <KeyOutlined />
          Edit Authentication
        </Space>
      }
      open={open}
      onCancel={onClose}
      width={560}
      okText="Save Changes"
      confirmLoading={loading}
      onOk={handleSave}
    >
      <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
        {/* Kerberos Section */}
        <Card size="small" title="Kerberos / SPNEGO" type="inner" style={{ marginBottom: 16 }}>
          <Form.Item
            name="kerberosEnabled"
            valuePropName="checked"
            style={{ marginBottom: 16 }}
          >
            <Switch
              checkedChildren="Enabled"
              unCheckedChildren="Disabled"
              onChange={(checked) => setKerberosEnabled(checked)}
            />
          </Form.Item>

          <Form.Item
            name="keytab"
            label="Keytab Path"
            rules={kerberosEnabled ? [{ required: true, message: 'Required when Kerberos is enabled' }] : []}
          >
            <Input
              placeholder="/etc/krb5.keytab"
              disabled={!kerberosEnabled}
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>

          <Form.Item
            name="spn"
            label="Service Principal Name (SPN)"
            tooltip="Optional — derived from the keytab. Managed via SPN Management tab."
          >
            <Input
              placeholder="HTTP/sambmin.example.com@EXAMPLE.COM"
              disabled={!kerberosEnabled}
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
            />
          </Form.Item>

          {kerberosEnabled && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              Implementation: <Text code>{data.kerberos.implementation}</Text> (read-only, set via config)
            </Text>
          )}
        </Card>

        {/* LDAP Bind Section */}
        <Card size="small" title="LDAP Bind" type="inner" style={{ marginBottom: 16 }}>
          <Form.Item
            name="ldapEnabled"
            valuePropName="checked"
            style={{ marginBottom: 0 }}
          >
            <Switch
              checkedChildren="Enabled"
              unCheckedChildren="Disabled"
              onChange={(checked) => setLdapEnabled(checked)}
            />
          </Form.Item>
          {!ldapEnabled && !kerberosEnabled && (
            <Text type="danger" style={{ fontSize: 12, display: 'block', marginTop: 8 }}>
              At least one authentication method must be enabled.
            </Text>
          )}
        </Card>

        {/* Session Timeout */}
        <Divider orientation="left" plain style={{ fontSize: 12 }}>Session</Divider>

        <Form.Item
          name="sessionTimeout"
          label="Session Timeout"
        >
          <Slider
            min={1}
            max={24}
            marks={sessionTimeoutMarks}
            tooltip={{
              formatter: (value) => `${value} hour${value === 1 ? '' : 's'}`,
            }}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
