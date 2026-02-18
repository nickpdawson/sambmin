import { useState } from 'react';
import {
  Drawer, Form, Input, Select, Button, Space,
  notification, Modal,
} from 'antd';
import { api } from '../../api/client';

interface CreateGroupDrawerProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function CreateGroupDrawer({ open, onClose, onSuccess }: CreateGroupDrawerProps) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await api.post('/groups', {
        name: values.name,
        description: values.description || '',
        groupType: values.groupType || 'Security',
        groupScope: values.groupScope || 'Global',
        ou: values.ou || '',
      });
      notification.success({
        message: 'Group created',
        description: values.name,
      });
      form.resetFields();
      onSuccess();
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'errorFields' in err) return;
      const msg = err instanceof Error ? err.message : 'Failed to create group';
      Modal.error({ title: 'Create group failed', content: msg });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Drawer
      title="Create Group"
      placement="right"
      width={480}
      open={open}
      onClose={() => { form.resetFields(); onClose(); }}
      extra={
        <Space>
          <Button onClick={() => { form.resetFields(); onClose(); }}>Cancel</Button>
          <Button type="primary" loading={loading} onClick={handleSubmit}>
            Create Group
          </Button>
        </Space>
      }
    >
      <Form form={form} layout="vertical" initialValues={{ groupType: 'Security', groupScope: 'Global' }}>
        <Form.Item
          name="name"
          label="Group Name"
          rules={[{ required: true, message: 'Group name is required' }]}
        >
          <Input placeholder="e.g. IT-Admins" />
        </Form.Item>

        <Form.Item name="description" label="Description">
          <Input.TextArea rows={2} placeholder="Optional description" />
        </Form.Item>

        <Space style={{ width: '100%' }} size={12}>
          <Form.Item name="groupType" label="Type" style={{ flex: 1 }}>
            <Select
              options={[
                { value: 'Security', label: 'Security' },
                { value: 'Distribution', label: 'Distribution' },
              ]}
            />
          </Form.Item>

          <Form.Item name="groupScope" label="Scope" style={{ flex: 1 }}>
            <Select
              options={[
                { value: 'Global', label: 'Global' },
                { value: 'DomainLocal', label: 'Domain Local' },
                { value: 'Universal', label: 'Universal' },
              ]}
            />
          </Form.Item>
        </Space>
      </Form>
    </Drawer>
  );
}
