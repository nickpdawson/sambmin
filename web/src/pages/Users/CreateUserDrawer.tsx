import { useState, useEffect } from 'react';
import {
  Drawer, Form, Input, Select, Switch, Button, Space, Collapse, Typography,
  notification, Modal,
} from 'antd';
import { UserOutlined, LockOutlined, ReloadOutlined } from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

interface CreateUserDrawerProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

function generatePassword(length = 16): string {
  const upper = 'ABCDEFGHJKLMNPQRSTUVWXYZ';
  const lower = 'abcdefghjkmnpqrstuvwxyz';
  const digits = '23456789';
  const special = '!@#$%&*';
  const all = upper + lower + digits + special;

  let password = '';
  // Ensure at least one of each type
  password += upper[Math.floor(Math.random() * upper.length)];
  password += lower[Math.floor(Math.random() * lower.length)];
  password += digits[Math.floor(Math.random() * digits.length)];
  password += special[Math.floor(Math.random() * special.length)];

  for (let i = password.length; i < length; i++) {
    password += all[Math.floor(Math.random() * all.length)];
  }

  // Shuffle
  return password.split('').sort(() => Math.random() - 0.5).join('');
}

interface OUOption { dn: string; name: string }
interface GroupOption { dn: string; name: string; samAccountName: string }

export default function CreateUserDrawer({ open, onClose, onSuccess }: CreateUserDrawerProps) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [ouOptions, setOUOptions] = useState<{ value: string; label: string }[]>([]);
  const [groupOptions, setGroupOptions] = useState<{ value: string; label: string }[]>([]);

  useEffect(() => {
    if (!open) return;
    api.get<{ ous: OUOption[] }>('/ous')
      .then((data) => setOUOptions(
        (data.ous || []).map((ou) => ({ value: ou.dn, label: ou.name }))
      ))
      .catch(() => {});
    api.get<{ groups: GroupOption[] }>('/groups')
      .then((data) => setGroupOptions(
        (data.groups || []).map((g) => ({ value: g.samAccountName || g.name, label: g.name }))
      ))
      .catch(() => {});
  }, [open]);

  const handleGenerate = () => {
    const pw = generatePassword();
    form.setFieldsValue({ password: pw, confirmPassword: pw });
  };

  const handleAutoName = () => {
    const first = form.getFieldValue('givenName') || '';
    const last = form.getFieldValue('surname') || '';
    if (first && last) {
      const username = (first[0] + last).toLowerCase().replace(/[^a-z0-9]/g, '');
      const display = `${first} ${last}`;
      form.setFieldsValue({
        samAccountName: username,
        displayName: display,
        mail: `${username}@dzsec.net`,
        userPrincipalName: `${username}@dzsec.net`,
      });
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await api.post('/users', {
        username: values.samAccountName,
        password: values.password,
        givenName: values.givenName,
        surname: values.surname,
        mail: values.mail,
        department: values.department,
        title: values.title,
        ou: values.ou,
        mustChangePassword: values.mustChangePassword ?? true,
      });
      notification.success({ message: 'User created successfully', description: values.displayName || values.samAccountName });
      form.resetFields();
      onSuccess();
    } catch (err: unknown) {
      // Show validation errors inline (Ant Design handles that), only show API errors
      if (err && typeof err === 'object' && 'errorFields' in err) return; // form validation — inline errors shown
      const msg = err instanceof Error ? err.message : 'Failed to create user';
      Modal.error({ title: 'Create user failed', content: msg });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Drawer
      title="Create User"
      placement="right"
      width={560}
      open={open}
      onClose={() => { form.resetFields(); onClose(); }}
      extra={
        <Space>
          <Button onClick={() => { form.resetFields(); onClose(); }}>Cancel</Button>
          <Button type="primary" loading={loading} onClick={handleSubmit}>
            Create User
          </Button>
        </Space>
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          mustChangePassword: true,
          enabled: true,
        }}
      >
        {/* Identity */}
        <Text strong style={{ display: 'block', marginBottom: 12 }}>Identity</Text>

        <Space style={{ width: '100%' }} size={12}>
          <Form.Item
            name="givenName"
            label="First Name"
            rules={[{ required: true, message: 'Required' }]}
            style={{ flex: 1 }}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="John"
              onBlur={handleAutoName}
            />
          </Form.Item>

          <Form.Item
            name="surname"
            label="Last Name"
            rules={[{ required: true, message: 'Required' }]}
            style={{ flex: 1 }}
          >
            <Input
              placeholder="Smith"
              onBlur={handleAutoName}
            />
          </Form.Item>
        </Space>

        <Form.Item name="displayName" label="Display Name">
          <Input placeholder="Auto-generated from first + last" />
        </Form.Item>

        <Space style={{ width: '100%' }} size={12}>
          <Form.Item
            name="samAccountName"
            label="Username"
            rules={[{ required: true, message: 'Required' }]}
            style={{ flex: 1 }}
          >
            <Input placeholder="jsmith" addonAfter="@dzsec.net" />
          </Form.Item>
        </Space>

        <Form.Item name="mail" label="Email">
          <Input placeholder="jsmith@dzsec.net" />
        </Form.Item>

        {/* Credentials */}
        <Text strong style={{ display: 'block', marginBottom: 12, marginTop: 8 }}>Credentials</Text>

        <Form.Item
          name="password"
          label="Password"
          rules={[
            { required: true, message: 'Required' },
            { min: 12, message: 'Must be at least 12 characters' },
          ]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="Minimum 12 characters"
            addonAfter={
              <Button type="link" size="small" icon={<ReloadOutlined />} onClick={handleGenerate} style={{ margin: -4 }}>
                Generate
              </Button>
            }
          />
        </Form.Item>

        <Form.Item
          name="confirmPassword"
          label="Confirm Password"
          dependencies={['password']}
          rules={[
            { required: true, message: 'Required' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('password') === value) return Promise.resolve();
                return Promise.reject(new Error('Passwords do not match'));
              },
            }),
          ]}
        >
          <Input.Password prefix={<LockOutlined />} placeholder="Confirm password" />
        </Form.Item>

        <Space size={24}>
          <Form.Item name="mustChangePassword" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Switch checkedChildren="Must change password" unCheckedChildren="No change required" />
          </Form.Item>

          <Form.Item name="enabled" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Switch checkedChildren="Account enabled" unCheckedChildren="Account disabled" />
          </Form.Item>
        </Space>

        {/* Organization — collapsible */}
        <Collapse
          ghost
          items={[
            {
              key: 'org',
              label: 'Organization',
              children: (
                <>
                  <Form.Item name="department" label="Department">
                    <Input placeholder="Engineering" />
                  </Form.Item>
                  <Form.Item name="title" label="Job Title">
                    <Input placeholder="Software Engineer" />
                  </Form.Item>
                  <Form.Item name="ou" label="Organizational Unit">
                    <Select
                      placeholder="Select OU..."
                      options={ouOptions}
                      showSearch
                      filterOption={(input, option) =>
                        (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                      }
                    />
                  </Form.Item>
                </>
              ),
            },
            {
              key: 'groups',
              label: 'Group Membership',
              children: (
                <Form.Item name="groups" label="Additional Groups">
                  <Select
                    mode="multiple"
                    placeholder="Search and add groups..."
                    options={groupOptions}
                    showSearch
                    filterOption={(input, option) =>
                      (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                    }
                  />
                </Form.Item>
              ),
            },
          ]}
          style={{ marginTop: 8 }}
        />

        {/* CLI Equivalent — collapsible */}
        <Collapse
          ghost
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
                }}>
{`samba-tool user create ${form.getFieldValue('samAccountName') || '<username>'} \\
  --given-name="${form.getFieldValue('givenName') || ''}" \\
  --surname="${form.getFieldValue('surname') || ''}" \\
  --mail-address="${form.getFieldValue('mail') || ''}" \\
  --must-change-at-next-login`}
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
