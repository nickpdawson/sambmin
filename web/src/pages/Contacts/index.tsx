import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Button, Input, Space, Tag, Dropdown,
  notification, Modal, Form,
} from 'antd';
import {
  PlusOutlined, ReloadOutlined, MoreOutlined,
  SearchOutlined, ExclamationCircleOutlined, ContactsOutlined,
} from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { api } from '../../api/client';
import ExportButton from '../../components/ExportButton';
import ContactDrawer from './ContactDrawer';


interface Contact {
  dn: string;
  name: string;
  displayName: string;
  givenName: string;
  sn: string;
  mail: string;
  description: string;
  department: string;
  title: string;
  company: string;
  office: string;
  phone: string;
  mobile: string;
  streetAddress: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
  whenCreated: string;
  whenChanged: string;
  memberOf: string[];
}

export default function Contacts() {
  const actionRef = useRef<ActionType>(null);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [selectedContact, setSelectedContact] = useState<Contact | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [renameTarget, setRenameTarget] = useState<Contact | null>(null);
  const [renameForm] = Form.useForm();

  const loadContacts = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ contacts: Contact[] }>('/contacts');
      setContacts(data.contacts);
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadContacts();
  }, [loadContacts]);

  const handleContactAction = useCallback(async (action: string, record: Contact) => {
    const dn = encodeURIComponent(record.dn);
    const name = record.displayName || record.name;

    if (action === 'delete') {
      Modal.confirm({
        title: 'Delete Contact',
        icon: <ExclamationCircleOutlined />,
        content: `Are you sure you want to delete ${name}? This cannot be undone.`,
        okText: 'Delete',
        okButtonProps: { danger: true },
        onOk: async () => {
          await api.delete(`/contacts/${dn}`);
          notification.success({ message: `${name} deleted` });
          loadContacts();
        },
      });
      return;
    }

    if (action === 'rename') {
      setRenameTarget(record);
      renameForm.setFieldsValue({ newName: record.name });
      return;
    }
  }, [loadContacts, renameForm]);

  const handleCreate = useCallback(async () => {
    try {
      const values = await createForm.validateFields();
      await api.post('/contacts', values);
      notification.success({ message: `Contact ${values.name} created` });
      createForm.resetFields();
      setCreateOpen(false);
      loadContacts();
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        Modal.error({ title: 'Failed to create contact', content: err.message });
      }
    }
  }, [createForm, loadContacts]);

  const handleRename = useCallback(async () => {
    if (!renameTarget) return;
    try {
      const values = await renameForm.validateFields();
      const dn = encodeURIComponent(renameTarget.dn);
      await api.post(`/contacts/${dn}/rename`, { newName: values.newName });
      notification.success({ message: `Renamed to ${values.newName}` });
      renameForm.resetFields();
      setRenameTarget(null);
      loadContacts();
    } catch (err: unknown) {
      if (err instanceof Error && err.message) {
        Modal.error({ title: 'Rename failed', content: err.message });
      }
    }
  }, [renameTarget, renameForm, loadContacts]);

  const filteredContacts = contacts.filter((c) => {
    if (search) {
      const s = search.toLowerCase();
      return (
        (c.displayName || '').toLowerCase().includes(s) ||
        (c.name || '').toLowerCase().includes(s) ||
        (c.mail || '').toLowerCase().includes(s) ||
        (c.company || '').toLowerCase().includes(s)
      );
    }
    return true;
  });

  const columns: ProColumns<Contact>[] = [
    {
      title: 'Name',
      dataIndex: 'displayName',
      key: 'displayName',
      sorter: (a, b) => (a.displayName || a.name).localeCompare(b.displayName || b.name),
      render: (_, record) => (
        <div>
          <a onClick={() => { setSelectedContact(record); setDrawerOpen(true); }}>
            <Space size={6}>
              <ContactsOutlined />
              {record.displayName || record.name}
            </Space>
          </a>
        </div>
      ),
    },
    {
      title: 'Email',
      dataIndex: 'mail',
      key: 'mail',
      copyable: true,
      ellipsis: true,
    },
    {
      title: 'Company',
      dataIndex: 'company',
      key: 'company',
      filters: [...new Set(contacts.map((c) => c.company).filter(Boolean))].map((co) => ({
        text: co, value: co,
      })),
      onFilter: (value, record) => record.company === value,
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
      responsive: ['lg'],
      ellipsis: true,
    },
    {
      title: 'Phone',
      dataIndex: 'phone',
      key: 'phone',
      responsive: ['xl'],
    },
    {
      title: 'Groups',
      key: 'groups',
      responsive: ['xl'],
      render: (_, record) => (
        <Space size={4} wrap>
          {(record.memberOf || []).slice(0, 2).map((g) => (
            <Tag key={g} style={{ fontSize: 11 }}>{g.split(',')[0]?.replace(/^CN=/i, '') || g}</Tag>
          ))}
          {(record.memberOf || []).length > 2 && (
            <Tag style={{ fontSize: 11 }}>+{record.memberOf.length - 2}</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '',
      key: 'actions',
      width: 48,
      render: (_, record) => (
        <Dropdown
          menu={{
            items: [
              { key: 'view', label: 'View Details' },
              { key: 'rename', label: 'Rename' },
              { type: 'divider' },
              { key: 'delete', label: 'Delete Contact', danger: true },
            ],
            onClick: ({ key }) => {
              if (key === 'view') {
                setSelectedContact(record);
                setDrawerOpen(true);
              } else {
                handleContactAction(key, record);
              }
            },
          }}
          trigger={['click']}
        >
          <Button type="text" icon={<MoreOutlined />} size="small" />
        </Dropdown>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <ProTable<Contact>
        actionRef={actionRef}
        columns={columns}
        dataSource={filteredContacts}
        rowKey="dn"
        loading={loading}
        search={false}
        dateFormatter="string"
        options={false}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `${total} contacts`,
        }}
        toolBarRender={() => [
          <Input
            key="search"
            placeholder="Search contacts..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            allowClear
            style={{ width: 240 }}
          />,
          <ExportButton
            key="export"
            data={filteredContacts as unknown as Record<string, unknown>[]}
            filename="sambmin-contacts"
            columns={[
              { key: 'name', title: 'Name' },
              { key: 'mail', title: 'Email' },
              { key: 'department', title: 'Department' },
              { key: 'title', title: 'Title' },
              { key: 'company', title: 'Company' },
              { key: 'phone', title: 'Phone' },
              { key: 'dn', title: 'DN' },
            ]}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={loadContacts} />,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            New Contact
          </Button>,
        ]}
      />

      <ContactDrawer
        contact={selectedContact}
        open={drawerOpen}
        onClose={() => { setDrawerOpen(false); setSelectedContact(null); }}
        onRefresh={async () => {
          await loadContacts();
          if (selectedContact) {
            try {
              const dn = encodeURIComponent(selectedContact.dn);
              const fresh = await api.get<Contact>(`/contacts/${dn}`);
              setSelectedContact(fresh);
            } catch { /* keep stale data */ }
          }
        }}
      />

      {/* Create Contact Modal */}
      <Modal
        title="New Contact"
        open={createOpen}
        onCancel={() => { createForm.resetFields(); setCreateOpen(false); }}
        onOk={handleCreate}
        okText="Create"
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="name" label="Full Name" rules={[{ required: true, message: 'Name is required' }]}>
            <Input placeholder="Jane Doe" />
          </Form.Item>
          <Form.Item name="givenName" label="First Name">
            <Input />
          </Form.Item>
          <Form.Item name="surname" label="Last Name">
            <Input />
          </Form.Item>
          <Form.Item name="mail" label="Email">
            <Input type="email" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      {/* Rename Contact Modal */}
      <Modal
        title={`Rename Contact — ${renameTarget?.displayName || renameTarget?.name || ''}`}
        open={!!renameTarget}
        onCancel={() => { renameForm.resetFields(); setRenameTarget(null); }}
        onOk={handleRename}
        okText="Rename"
      >
        <Form form={renameForm} layout="vertical">
          <Form.Item name="newName" label="New Name" rules={[{ required: true, message: 'New name is required' }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
