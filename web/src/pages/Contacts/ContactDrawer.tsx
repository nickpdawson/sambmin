import { useState, useCallback } from 'react';
import {
  Drawer, Descriptions, Space, Button, Typography, Tooltip, Tabs,
  notification, Input,
} from 'antd';
import {
  ContactsOutlined, CopyOutlined, EditOutlined, SaveOutlined, CloseOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';

const { Text } = Typography;

const mono: React.CSSProperties = { fontFamily: '"JetBrains Mono", monospace' };

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

interface ContactDrawerProps {
  contact: Contact | null;
  open: boolean;
  onClose: () => void;
  onRefresh?: () => void;
}

function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text);
  notification.success({ message: `${label} copied`, duration: 2, placement: 'bottomRight' });
}

const cnFromDN = (dn: string) => dn.split(',')[0]?.replace(/^CN=/i, '') || dn;

// Inline editable field
function EditableField({
  label, value, fieldKey, dn, onSaved,
}: {
  label: string;
  value: string;
  fieldKey: string;
  dn: string;
  onSaved: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(value);
  const [saving, setSaving] = useState(false);

  const save = useCallback(async () => {
    setSaving(true);
    try {
      await api.put(`/contacts/${encodeURIComponent(dn)}`, { [fieldKey]: editValue });
      notification.success({ message: `${label} updated`, duration: 2 });
      setEditing(false);
      onSaved();
    } catch (err: unknown) {
      notification.error({ message: `Failed to update ${label}`, description: err instanceof Error ? err.message : undefined });
    } finally {
      setSaving(false);
    }
  }, [dn, fieldKey, editValue, label, onSaved]);

  if (editing) {
    return (
      <Space>
        <Input
          size="small"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onPressEnter={save}
          style={{ width: 200 }}
        />
        <Button type="text" size="small" icon={<SaveOutlined />} loading={saving} onClick={save} />
        <Button type="text" size="small" icon={<CloseOutlined />} onClick={() => { setEditing(false); setEditValue(value); }} />
      </Space>
    );
  }

  return (
    <Space>
      <Text>{value || <Text type="secondary">—</Text>}</Text>
      <Tooltip title={`Edit ${label}`}>
        <Button type="text" size="small" icon={<EditOutlined />} onClick={() => { setEditValue(value); setEditing(true); }} />
      </Tooltip>
    </Space>
  );
}

export default function ContactDrawer({ contact, open, onClose, onRefresh }: ContactDrawerProps) {
  if (!contact) return null;

  const refresh = () => onRefresh?.();

  const tabItems = [
    {
      key: 'identity',
      label: 'Identity',
      children: (
        <>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="Display Name">
              <EditableField label="Display Name" value={contact.displayName} fieldKey="displayName" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
            <Descriptions.Item label="First Name">
              <EditableField label="First Name" value={contact.givenName} fieldKey="givenName" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
            <Descriptions.Item label="Last Name">
              <EditableField label="Last Name" value={contact.sn} fieldKey="surname" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
            <Descriptions.Item label="Email">
              <EditableField label="Email" value={contact.mail} fieldKey="mail" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
            <Descriptions.Item label="Description">
              <EditableField label="Description" value={contact.description} fieldKey="description" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
            <Descriptions.Item label="Phone">
              <EditableField label="Phone" value={contact.phone} fieldKey="phone" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
            <Descriptions.Item label="Mobile">
              <EditableField label="Mobile" value={contact.mobile} fieldKey="mobile" dn={contact.dn} onSaved={refresh} />
            </Descriptions.Item>
          </Descriptions>
          <div style={{ marginTop: 16 }}>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="DN">
                <Space>
                  <Text style={{ fontSize: 12, wordBreak: 'break-all', ...mono }}>{contact.dn}</Text>
                  <Tooltip title="Copy DN">
                    <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(contact.dn, 'DN')} />
                  </Tooltip>
                </Space>
              </Descriptions.Item>
            </Descriptions>
          </div>
        </>
      ),
    },
    {
      key: 'organization',
      label: 'Organization',
      children: (
        <Descriptions column={1} size="small" bordered>
          <Descriptions.Item label="Title">
            <EditableField label="Title" value={contact.title} fieldKey="title" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="Department">
            <EditableField label="Department" value={contact.department} fieldKey="department" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="Company">
            <EditableField label="Company" value={contact.company} fieldKey="company" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="Office">
            <EditableField label="Office" value={contact.office} fieldKey="office" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="Street">
            <EditableField label="Street" value={contact.streetAddress} fieldKey="streetAddress" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="City">
            <EditableField label="City" value={contact.city} fieldKey="city" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="State">
            <EditableField label="State" value={contact.state} fieldKey="state" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="Postal Code">
            <EditableField label="Postal Code" value={contact.postalCode} fieldKey="postalCode" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
          <Descriptions.Item label="Country">
            <EditableField label="Country" value={contact.country} fieldKey="country" dn={contact.dn} onSaved={refresh} />
          </Descriptions.Item>
        </Descriptions>
      ),
    },
    {
      key: 'groups',
      label: `Groups (${(contact.memberOf || []).length})`,
      children: (
        <>
          {(contact.memberOf || []).length > 0 ? (
            <Descriptions column={1} size="small" bordered>
              {contact.memberOf.map((groupDN) => (
                <Descriptions.Item key={groupDN} label={cnFromDN(groupDN)}>
                  <Text style={{ fontSize: 12, ...mono }}>{groupDN}</Text>
                </Descriptions.Item>
              ))}
            </Descriptions>
          ) : (
            <Text type="secondary">Not a member of any groups</Text>
          )}
        </>
      ),
    },
    {
      key: 'details',
      label: 'Details',
      children: (
        <Descriptions column={1} size="small" bordered>
          <Descriptions.Item label="Created">
            {contact.whenCreated ? new Date(contact.whenCreated).toLocaleString() : 'Unknown'}
          </Descriptions.Item>
          <Descriptions.Item label="Modified">
            {contact.whenChanged ? new Date(contact.whenChanged).toLocaleString() : 'Unknown'}
          </Descriptions.Item>
        </Descriptions>
      ),
    },
  ];

  return (
    <Drawer
      title={
        <Space>
          <ContactsOutlined />
          <span>{contact.displayName || contact.name}</span>
        </Space>
      }
      placement="right"
      width={560}
      open={open}
      onClose={onClose}
    >
      <Tabs items={tabItems} />
    </Drawer>
  );
}
