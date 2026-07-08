import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Card, Select, Button, Space, Typography, Tag, Checkbox, Alert, Table,
  notification, Modal, Empty, Divider, Tooltip,
} from 'antd';
import {
  SafetyCertificateOutlined, ReloadOutlined, TeamOutlined, UserOutlined,
  DesktopOutlined, ApartmentOutlined, DeleteOutlined, ExclamationCircleOutlined,
} from '@ant-design/icons';
import { api } from '../../api/client';
import { useAuth } from '../../hooks/useAuth';

const { Title, Text, Paragraph } = Typography;

interface Template {
  key: string;
  label: string;
  description: string;
  category: string;
  risk: 'low' | 'medium' | 'high';
  appliesTo: string;
}

interface OU { dn: string; name: string }
interface UserRow { dn: string; samAccountName: string; displayName: string }
interface GroupRow { dn: string; name: string; samAccountName: string }

interface AclEntry {
  trusteeSid: string;
  trusteeName: string;
  trusteeClass?: string;
  rights: string;
  templateKey?: string;
  templateLabel?: string;
  rawAce: string;
}

interface ApplyResult {
  trusteeDn: string;
  trusteeName: string;
  templateKey: string;
  templateLabel: string;
  ok: boolean;
  error?: string;
}

const mono: React.CSSProperties = { fontFamily: '"JetBrains Mono", monospace', fontSize: 12 };

function riskTag(risk: string) {
  const color = risk === 'high' ? 'red' : risk === 'medium' ? 'gold' : 'green';
  return <Tag color={color} style={{ marginInlineStart: 8 }}>{risk} risk</Tag>;
}

function trusteeIcon(cls?: string) {
  if (cls === 'group') return <TeamOutlined />;
  if (cls === 'computer') return <DesktopOutlined />;
  return <UserOutlined />;
}

// Convert a dotted domain (alpinenet.us) to its DN form (DC=alpinenet,DC=us).
function domainToDN(domain: string): string {
  if (!domain) return '';
  return domain.split('.').map((p) => `DC=${p}`).join(',');
}

export default function DelegationPage() {
  const { user } = useAuth();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [ous, setOUs] = useState<OU[]>([]);
  const [trusteeOptions, setTrusteeOptions] = useState<{ value: string; label: string; cls: string }[]>([]);

  const domainRootDN = useMemo(() => {
    if (user?.domain) return domainToDN(user.domain);
    // Fallback: derive from any OU DN.
    const withDC = ous.find((o) => o.dn.includes('DC='));
    return withDC ? withDC.dn.slice(withDC.dn.indexOf('DC=')) : '';
  }, [user?.domain, ous]);

  const [target, setTarget] = useState<string>('');
  const [selectedTrustees, setSelectedTrustees] = useState<string[]>([]);
  const [selectedTemplates, setSelectedTemplates] = useState<string[]>([]);
  const [applying, setApplying] = useState(false);

  const [entries, setEntries] = useState<AclEntry[]>([]);
  const [loadingAcl, setLoadingAcl] = useState(false);

  // Initial load: templates, OUs, trustees.
  useEffect(() => {
    api.get<{ templates: Template[] }>('/dsacl/templates')
      .then((d) => setTemplates(d.templates || []))
      .catch(() => {});
    api.get<{ ous: OU[] }>('/ous')
      .then((d) => setOUs(d.ous || []))
      .catch(() => {});
    Promise.all([
      api.get<{ users: UserRow[] }>('/users').catch(() => ({ users: [] })),
      api.get<{ groups: GroupRow[] }>('/groups').catch(() => ({ groups: [] })),
    ]).then(([u, g]) => {
      const opts = [
        ...(u.users || []).map((x) => ({
          value: x.dn, label: `${x.displayName || x.samAccountName} (user)`, cls: 'user',
        })),
        ...(g.groups || []).map((x) => ({
          value: x.dn, label: `${x.name} (group)`, cls: 'group',
        })),
      ].sort((a, b) => a.label.localeCompare(b.label));
      setTrusteeOptions(opts);
    });
  }, []);

  // Default the target to the domain root once known.
  useEffect(() => {
    if (!target && domainRootDN) setTarget(domainRootDN);
  }, [domainRootDN, target]);

  const targetOptions = useMemo(() => {
    const opts: { value: string; label: string }[] = [];
    if (domainRootDN) opts.push({ value: domainRootDN, label: `Domain root — ${domainRootDN}` });
    for (const o of ous) opts.push({ value: o.dn, label: o.dn });
    return opts;
  }, [domainRootDN, ous]);

  const loadAcl = useCallback(async (dn: string) => {
    if (!dn) { setEntries([]); return; }
    setLoadingAcl(true);
    try {
      const d = await api.get<{ entries: AclEntry[] }>(`/dsacl?objectDn=${encodeURIComponent(dn)}`);
      setEntries(d.entries || []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to read delegations';
      notification.error({ message: 'Could not read delegations', description: msg });
      setEntries([]);
    } finally {
      setLoadingAcl(false);
    }
  }, []);

  useEffect(() => { loadAcl(target); }, [target, loadAcl]);

  const templatesByCategory = useMemo(() => {
    const map = new Map<string, Template[]>();
    for (const t of templates) {
      if (!map.has(t.category)) map.set(t.category, []);
      map.get(t.category)!.push(t);
    }
    return [...map.entries()];
  }, [templates]);

  const selectedTemplateObjs = templates.filter((t) => selectedTemplates.includes(t.key));
  const anyHighRisk = selectedTemplateObjs.some((t) => t.risk === 'high');
  const anyReplication = selectedTemplateObjs.some((t) => t.category === 'Directory replication');
  const targetIsDomainRoot = target === domainRootDN;

  const handleApply = async () => {
    if (!target || selectedTrustees.length === 0 || selectedTemplates.length === 0) return;
    const doApply = async () => {
      setApplying(true);
      try {
        const res = await api.post<{ results: ApplyResult[]; applied: number; failed: number }>('/dsacl/apply', {
          objectDn: target,
          trustees: selectedTrustees,
          templates: selectedTemplates,
        });
        if (res.failed > 0) {
          const failures = res.results.filter((r) => !r.ok)
            .map((r) => `${r.trusteeName || r.trusteeDn} → ${r.templateLabel}: ${r.error}`);
          notification.warning({
            message: `Applied ${res.applied}, ${res.failed} failed`,
            description: <ul style={{ margin: 0, paddingInlineStart: 18 }}>{failures.map((f, i) => <li key={i}>{f}</li>)}</ul>,
            duration: 0,
          });
        } else {
          notification.success({ message: `Granted ${res.applied} delegation${res.applied === 1 ? '' : 's'}` });
        }
        setSelectedTrustees([]);
        setSelectedTemplates([]);
        loadAcl(target);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : 'Apply failed';
        Modal.error({ title: 'Delegation failed', content: msg });
      } finally {
        setApplying(false);
      }
    };

    if (anyHighRisk) {
      Modal.confirm({
        title: 'Grant high-privilege delegation?',
        icon: <ExclamationCircleOutlined />,
        content: (
          <div>
            <Paragraph>You are about to grant high-risk capabilities:</Paragraph>
            <ul>
              {selectedTemplateObjs.filter((t) => t.risk === 'high').map((t) => <li key={t.key}>{t.label}</li>)}
            </ul>
            <Paragraph type="secondary">
              To {selectedTrustees.length} trustee{selectedTrustees.length === 1 ? '' : 's'} on <span style={mono}>{target}</span>.
            </Paragraph>
          </div>
        ),
        okText: 'Grant',
        okButtonProps: { danger: true },
        onOk: doApply,
      });
    } else {
      doApply();
    }
  };

  // Group ACL entries by trustee + capability so a multi-ACE delegation shows once.
  const delegationRows = useMemo(() => {
    const map = new Map<string, {
      key: string; trusteeSid: string; trusteeName: string; trusteeClass?: string;
      capability: string; rawAces: string[];
    }>();
    for (const e of entries) {
      const capKey = e.templateKey || e.rights;
      const key = `${e.trusteeSid}|${capKey}`;
      if (!map.has(key)) {
        map.set(key, {
          key, trusteeSid: e.trusteeSid, trusteeName: e.trusteeName, trusteeClass: e.trusteeClass,
          capability: e.templateLabel || e.rights, rawAces: [],
        });
      }
      map.get(key)!.rawAces.push(e.rawAce);
    }
    return [...map.values()];
  }, [entries]);

  const handleRemove = (row: { trusteeName: string; capability: string; rawAces: string[] }) => {
    Modal.confirm({
      title: 'Remove delegation',
      icon: <ExclamationCircleOutlined />,
      content: <span>Remove <b>{row.capability}</b> from <b>{row.trusteeName}</b> on this object?</span>,
      okText: 'Remove',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await api.post('/dsacl/remove', { objectDn: target, sddls: row.rawAces });
          notification.success({ message: 'Delegation removed' });
          loadAcl(target);
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : 'Remove failed';
          Modal.error({ title: 'Remove failed', content: msg });
        }
      },
    });
  };

  const delegationColumns = [
    {
      title: 'Trustee', key: 'trustee',
      render: (_: unknown, r: typeof delegationRows[number]) => (
        <Space>
          {trusteeIcon(r.trusteeClass)}
          <span>{r.trusteeName}</span>
          {r.trusteeClass && <Tag>{r.trusteeClass}</Tag>}
        </Space>
      ),
    },
    { title: 'Capability', dataIndex: 'capability', key: 'capability' },
    {
      title: '', key: 'actions', align: 'right' as const, width: 120,
      render: (_: unknown, r: typeof delegationRows[number]) => (
        <Button danger type="text" size="small" icon={<DeleteOutlined />} onClick={() => handleRemove(r)}>
          Remove
        </Button>
      ),
    },
  ];

  const canApply = !!target && selectedTrustees.length > 0 && selectedTemplates.length > 0;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          <SafetyCertificateOutlined style={{ marginRight: 8 }} />
          Delegation of Control
        </Title>
        <Button icon={<ReloadOutlined />} onClick={() => loadAcl(target)}>Refresh</Button>
      </div>

      <Paragraph type="secondary" style={{ maxWidth: 760 }}>
        Grant specific administrative rights on an OU (or the whole domain) to users and groups — the
        building blocks for service accounts, bind/sync accounts, OU administrators, and help-desk delegates.
        Select a target, one or more trustees, and one or more capabilities to grant them all at once.
      </Paragraph>

      <Card title="Grant delegation" style={{ marginBottom: 16 }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Text strong style={{ display: 'block', marginBottom: 6 }}>
              <ApartmentOutlined /> Target object
            </Text>
            <Select
              showSearch
              value={target || undefined}
              onChange={setTarget}
              options={targetOptions}
              placeholder="Select an OU or the domain root"
              style={{ width: '100%', maxWidth: 640 }}
              optionFilterProp="label"
            />
          </div>

          <div>
            <Text strong style={{ display: 'block', marginBottom: 6 }}>Trustees (who receives the rights)</Text>
            <Select
              mode="multiple"
              value={selectedTrustees}
              onChange={setSelectedTrustees}
              options={trusteeOptions}
              placeholder="Search users and groups…"
              style={{ width: '100%', maxWidth: 640 }}
              optionFilterProp="label"
              maxTagCount="responsive"
            />
          </div>

          <div>
            <Text strong style={{ display: 'block', marginBottom: 6 }}>Capabilities to grant</Text>
            <Checkbox.Group value={selectedTemplates} onChange={(v) => setSelectedTemplates(v as string[])} style={{ width: '100%' }}>
              {templatesByCategory.map(([category, items]) => (
                <div key={category} style={{ marginBottom: 12 }}>
                  <Text type="secondary" style={{ fontSize: 12, textTransform: 'uppercase', letterSpacing: 0.5 }}>{category}</Text>
                  <div style={{ marginTop: 4 }}>
                    {items.map((t) => (
                      <div key={t.key} style={{ padding: '4px 0' }}>
                        <Checkbox value={t.key}>
                          <span>{t.label}</span>
                          {riskTag(t.risk)}
                          {t.appliesTo === 'Domain root' && (
                            <Tag color="blue" style={{ marginInlineStart: 4 }}>domain root</Tag>
                          )}
                        </Checkbox>
                        <div style={{ marginInlineStart: 24, color: 'var(--ant-color-text-secondary)', fontSize: 12 }}>
                          {t.description}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </Checkbox.Group>
          </div>

          {anyReplication && !targetIsDomainRoot && (
            <Alert
              type="warning" showIcon
              message="Directory replication rights should target the domain root"
              description="You selected a replication capability but the target is an OU. These rights only take effect when applied on the domain root."
            />
          )}

          <Space>
            <Button type="primary" size="large" loading={applying} disabled={!canApply} onClick={handleApply}>
              Grant to {selectedTrustees.length || 0} trustee{selectedTrustees.length === 1 ? '' : 's'}
            </Button>
            {anyHighRisk && <Text type="danger"><ExclamationCircleOutlined /> Includes high-privilege rights</Text>}
          </Space>
        </Space>
      </Card>

      <Card
        title={
          <Space>
            Current delegations
            {target && <Text style={mono} type="secondary">{target}</Text>}
          </Space>
        }
      >
        {delegationRows.length === 0 && !loadingAcl ? (
          <Empty description="No delegations set on this object" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            loading={loadingAcl}
            dataSource={delegationRows}
            columns={delegationColumns}
            rowKey="key"
            size="small"
            pagination={false}
          />
        )}
        <Divider style={{ margin: '12px 0' }} />
        <Text type="secondary" style={{ fontSize: 12 }}>
          <Tooltip title="Only rights explicitly delegated to domain users and groups are shown; inherited defaults and built-in ACEs are hidden.">
            Showing delegations explicitly set on this object.
          </Tooltip>
        </Text>
      </Card>
    </div>
  );
}
