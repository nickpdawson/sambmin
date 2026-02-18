import { useState, useEffect } from 'react';
import {
  Modal, Button, Space, Typography, Descriptions, Tag, Switch, Upload, Form,
  Input, Divider, notification, Alert,
} from 'antd';
import {
  UploadOutlined, SafetyCertificateOutlined, ReloadOutlined,
  CheckCircleOutlined, WarningOutlined, InboxOutlined,
} from '@ant-design/icons';

const { Text } = Typography;
const { Dragger } = Upload;

interface TlsData {
  provider: string;
  domain: string;
  certificate: string;
  key: string;
  expiry: string;
  autoRenew: boolean;
}

interface TlsModalProps {
  open: boolean;
  data: TlsData;
  onClose: () => void;
  onSave: (data: TlsData) => void;
}

type View = 'details' | 'upload';

export default function TlsModal({ open, data, onClose, onSave }: TlsModalProps) {
  const [autoRenew, setAutoRenew] = useState(data.autoRenew);
  const [renewLoading, setRenewLoading] = useState(false);
  const [view, setView] = useState<View>('details');
  const [uploadForm] = Form.useForm();
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    if (open) {
      setAutoRenew(data.autoRenew);
      setView('details');
      uploadForm.resetFields();
    }
  }, [open, data, uploadForm]);

  const daysUntilExpiry = Math.ceil(
    (new Date(data.expiry).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
  );

  const handleRenewNow = async () => {
    setRenewLoading(true);
    // Simulate Let's Encrypt renewal
    await new Promise((r) => setTimeout(r, 2000));
    setRenewLoading(false);
    notification.success({
      message: 'Certificate Renewed',
      description: "Let's Encrypt certificate has been renewed successfully.",
      icon: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    });
  };

  const handleAutoRenewToggle = (checked: boolean) => {
    setAutoRenew(checked);
    onSave({ ...data, autoRenew: checked });
    notification.success({
      message: `Auto-renew ${checked ? 'enabled' : 'disabled'}`,
    });
  };

  const handleUpload = async () => {
    try {
      await uploadForm.validateFields();
      setUploading(true);
      // Simulate upload
      await new Promise((r) => setTimeout(r, 1000));
      setUploading(false);
      notification.success({
        message: 'Certificate Uploaded',
        description: 'Custom certificate has been installed.',
      });
      setView('details');
      onClose();
    } catch {
      setUploading(false);
    }
  };

  const uploadProps = {
    beforeUpload: () => false, // Prevent auto upload
    maxCount: 1,
    accept: '.pem,.crt,.cer',
  };

  const keyUploadProps = {
    beforeUpload: () => false,
    maxCount: 1,
    accept: '.pem,.key',
  };

  return (
    <Modal
      title={
        <Space>
          <SafetyCertificateOutlined />
          TLS Certificate Management
        </Space>
      }
      open={open}
      onCancel={onClose}
      width={600}
      footer={
        view === 'details' ? (
          <Space>
            <Button onClick={onClose}>Close</Button>
          </Space>
        ) : (
          <Space>
            <Button onClick={() => setView('details')}>Back</Button>
            <Button type="primary" loading={uploading} onClick={handleUpload}>
              Upload Certificate
            </Button>
          </Space>
        )
      }
    >
      {view === 'details' ? (
        <div style={{ marginTop: 8 }}>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Provider">
              <Tag color="green">
                {data.provider === 'letsencrypt' ? "Let's Encrypt" : data.provider}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Domain">
              <Text style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}>
                {data.domain}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="Certificate Path">
              <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                {data.certificate}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="Key Path">
              <Text copyable style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}>
                {data.key}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="Expiry">
              <Space>
                {daysUntilExpiry > 30 ? (
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                ) : (
                  <WarningOutlined style={{ color: '#faad14' }} />
                )}
                <Text>{new Date(data.expiry).toLocaleDateString()}</Text>
                <Text type="secondary">({daysUntilExpiry} days remaining)</Text>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="Auto-Renew">
              <Switch
                checked={autoRenew}
                onChange={handleAutoRenewToggle}
                checkedChildren="On"
                unCheckedChildren="Off"
              />
            </Descriptions.Item>
          </Descriptions>

          <Divider />

          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Button
              icon={<ReloadOutlined />}
              loading={renewLoading}
              onClick={handleRenewNow}
              block
            >
              Renew Now (Let's Encrypt)
            </Button>
            <Button
              icon={<UploadOutlined />}
              onClick={() => setView('upload')}
              block
            >
              Upload Custom Certificate
            </Button>
          </Space>
        </div>
      ) : (
        <div style={{ marginTop: 8 }}>
          <Alert
            message="Upload Custom Certificate"
            description="Upload a PEM-encoded certificate and private key. This will replace the current certificate."
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />

          <Form form={uploadForm} layout="vertical">
            <Form.Item
              name="certificate"
              label="Certificate File (.pem, .crt)"
              rules={[{ required: true, message: 'Certificate file is required' }]}
            >
              <Dragger {...uploadProps}>
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">Click or drag certificate file</p>
                <p className="ant-upload-hint">PEM-encoded certificate (.pem, .crt, .cer)</p>
              </Dragger>
            </Form.Item>

            <Form.Item
              name="privateKey"
              label="Private Key File (.pem, .key)"
              rules={[{ required: true, message: 'Private key file is required' }]}
            >
              <Dragger {...keyUploadProps}>
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">Click or drag private key file</p>
                <p className="ant-upload-hint">PEM-encoded private key (.pem, .key)</p>
              </Dragger>
            </Form.Item>

            <Form.Item name="chain" label="CA Chain (optional)">
              <Input.TextArea
                placeholder="Paste intermediate CA certificate chain here (PEM format)"
                rows={4}
                style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 12 }}
              />
            </Form.Item>
          </Form>
        </div>
      )}
    </Modal>
  );
}
