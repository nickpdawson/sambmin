import { useState, useEffect } from 'react';
import {
  Typography, Table, Tag, Card, Space, Button, Alert, Modal, Input,
  notification, Descriptions, Row, Col,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CrownOutlined, ReloadOutlined, SwapOutlined, WarningOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Title, Text } = Typography;
const mono = { fontFamily: "'JetBrains Mono', monospace", fontSize: 13 };

interface FSMORole {
  role: string;
  holder: string;
  dc: string;
}

const roleDescriptions: Record<string, string> = {
  'Schema Master': 'Controls schema modifications for the forest. Only one per forest.',
  'Domain Naming Master': 'Controls addition/removal of domains in the forest.',
  'PDC Emulator': 'Password changes, time sync, account lockout. Critical for auth.',
  'RID Master': 'Allocates RID pools to DCs for creating security principals.',
  'Infrastructure Master': 'Updates cross-domain group membership references.',
  'Domain DNS Zones Master': 'Controls the DomainDnsZones partition.',
  'Forest DNS Zones Master': 'Controls the ForestDnsZones partition.',
};

const roleAPINames: Record<string, string> = {
  'Schema Master': 'schema',
  'Domain Naming Master': 'naming',
  'PDC Emulator': 'pdc',
  'RID Master': 'rid',
  'Infrastructure Master': 'infrastructure',
  'Domain DNS Zones Master': 'domaindns',
  'Forest DNS Zones Master': 'forestdns',
};

export default function FSMO() {
  const [roles, setRoles] = useState<FSMORole[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [transferOpen, setTransferOpen] = useState(false);
  const [transferRole, setTransferRole] = useState('');
  const [confirmText, setConfirmText] = useState('');
  const [transferring, setTransferring] = useState(false);

  const fetchRoles = () => {
    setLoading(true);
    setError(null);
    api.get<{ roles: FSMORole[] }>('/fsmo')
      .then((data) => setRoles(data.roles || []))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchRoles(); }, []);

  const handleTransfer = async () => {
    if (!transferRole || confirmText !== 'TRANSFER') return;
    setTransferring(true);
    try {
      const apiRole = roleAPINames[transferRole] || transferRole.toLowerCase();
      await api.post('/fsmo/transfer', { role: apiRole });
      notification.success({
        message: 'FSMO transfer initiated',
        description: `${transferRole} transfer was successful`,
      });
      setTransferOpen(false);
      setTransferRole('');
      setConfirmText('');
      fetchRoles();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Transfer failed';
      notification.error({ message: 'Transfer failed', description: msg });
    } finally {
      setTransferring(false);
    }
  };

  // Count unique DCs
  const uniqueDCs = new Set(roles.map((r) => r.dc).filter(Boolean));

  const columns: ColumnsType<FSMORole> = [
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      width: 260,
      render: (role: string) => (
        <Space direction="vertical" size={0}>
          <Space>
            <CrownOutlined style={{ color: '#faad14' }} />
            <Text strong>{role}</Text>
          </Space>
          {roleDescriptions[role] && (
            <Text type="secondary" style={{ fontSize: 11, marginLeft: 22 }}>
              {roleDescriptions[role]}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: 'Current Holder (DC)',
      dataIndex: 'dc',
      key: 'dc',
      width: 200,
      render: (dc: string) => (
        <Tag color="blue">
          <Text style={{ ...mono, fontSize: 12 }}>{dc || 'Unknown'}</Text>
        </Tag>
      ),
    },
    {
      title: 'Full DN',
      dataIndex: 'holder',
      key: 'holder',
      ellipsis: true,
      render: (holder: string) => (
        <Text copyable style={{ ...mono, fontSize: 11 }}>{holder}</Text>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 100,
      render: (_: unknown, record: FSMORole) => (
        <Button
          size="small"
          icon={<SwapOutlined />}
          onClick={(e) => {
            e.stopPropagation();
            setTransferRole(record.role);
            setTransferOpen(true);
          }}
        >
          Transfer
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space align="center">
          <CrownOutlined style={{ fontSize: 20, color: '#faad14' }} />
          <Title level={4} style={{ margin: 0 }}>FSMO Roles</Title>
        </Space>
        <Button icon={<ReloadOutlined />} onClick={fetchRoles}>Refresh</Button>
      </div>

      {error && (
        <Alert type="error" message="Failed to load FSMO roles" description={error} style={{ marginBottom: 16 }} />
      )}

      <Alert
        type="info"
        showIcon
        message="FSMO Role Management"
        description="Flexible Single Master Operations roles ensure certain directory operations are performed by a single DC. Transfer roles carefully — the target DC must be reachable."
        style={{ marginBottom: 16 }}
        closable
      />

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Total Roles">{roles.length}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Unique Holders">{uniqueDCs.size}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="DCs">{Array.from(uniqueDCs).join(', ') || '---'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
      </Row>

      <Table
        columns={columns}
        dataSource={roles}
        rowKey="role"
        loading={loading}
        pagination={false}
        size="middle"
      />

      {/* CLI equivalent */}
      <Card size="small" style={{ marginTop: 16 }}>
        <Space direction="vertical" size={4}>
          <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalents:</Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool fsmo show
          </Text>
          <Text copyable style={{ ...mono, fontSize: 12 }}>
            samba-tool fsmo transfer --role=schema
          </Text>
        </Space>
      </Card>

      {/* Transfer Modal with type-to-confirm */}
      <Modal
        title={
          <Space>
            <WarningOutlined style={{ color: '#faad14' }} />
            <span>Transfer FSMO Role</span>
          </Space>
        }
        open={transferOpen}
        onCancel={() => { setTransferOpen(false); setTransferRole(''); setConfirmText(''); }}
        onOk={handleTransfer}
        okText="Transfer Role"
        okButtonProps={{ danger: true, disabled: confirmText !== 'TRANSFER' }}
        confirmLoading={transferring}
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Alert
            type="warning"
            message="This is a high-impact operation"
            description="Transferring FSMO roles changes which domain controller handles critical operations. Only proceed if you understand the implications."
          />

          <div>
            <Text strong>Role: </Text>
            <Tag color="orange">{transferRole}</Tag>
          </div>

          {roleDescriptions[transferRole] && (
            <Text type="secondary">{roleDescriptions[transferRole]}</Text>
          )}

          <div>
            <Text>Type <Text strong code>TRANSFER</Text> to confirm:</Text>
            <Input
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder="TRANSFER"
              style={{ marginTop: 8, ...mono }}
            />
          </div>

          <Card size="small" styles={{ body: { padding: '8px 12px' } }}>
            <Text type="secondary" style={{ fontSize: 12 }}>CLI equivalent: </Text>
            <Text copyable style={{ ...mono, fontSize: 12 }}>
              samba-tool fsmo transfer --role={roleAPINames[transferRole] || 'schema'}
            </Text>
          </Card>
        </Space>
      </Modal>
    </div>
  );
}
