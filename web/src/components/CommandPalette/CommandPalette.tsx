import { useEffect, useState, useCallback } from 'react';
import { Command } from 'cmdk';
import { useNavigate } from 'react-router-dom';
import {
  DashboardOutlined, UserOutlined, TeamOutlined, DesktopOutlined,
  ApartmentOutlined, GlobalOutlined, CloudServerOutlined,
  NodeIndexOutlined, SafetyCertificateOutlined, KeyOutlined,
  CrownOutlined, DatabaseOutlined, AuditOutlined, SettingOutlined,
  PlusOutlined, SearchOutlined, UnlockOutlined,
} from '@ant-design/icons';
import './CommandPalette.css';

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isAdmin: boolean;
}

export default function CommandPalette({ open, onOpenChange, isAdmin }: CommandPaletteProps) {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');

  const runAction = useCallback((action: () => void) => {
    action();
    onOpenChange(false);
    setSearch('');
  }, [onOpenChange]);

  // Close on escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onOpenChange(false);
        setSearch('');
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onOpenChange]);

  if (!open) return null;

  return (
    <div className="command-palette-overlay" onClick={() => { onOpenChange(false); setSearch(''); }}>
      <div className="command-palette-container" onClick={(e) => e.stopPropagation()}>
        <Command label="Command palette" shouldFilter={true}>
          <div className="command-palette-input-wrapper">
            <SearchOutlined className="command-palette-search-icon" />
            <Command.Input
              value={search}
              onValueChange={setSearch}
              placeholder="Search or type a command..."
              className="command-palette-input"
              autoFocus
            />
            <kbd className="command-palette-kbd">ESC</kbd>
          </div>

          <Command.List className="command-palette-list">
            <Command.Empty className="command-palette-empty">
              No results found.
            </Command.Empty>

            {/* Actions — admin only for write operations */}
            {isAdmin && (
              <Command.Group heading="Actions" className="command-palette-group">
                <Command.Item
                  value="Create new user"
                  onSelect={() => runAction(() => navigate('/users?action=create'))}
                  className="command-palette-item"
                >
                  <PlusOutlined />
                  <span>Create new user</span>
                </Command.Item>
                <Command.Item
                  value="Reset password"
                  onSelect={() => runAction(() => navigate('/users?action=reset'))}
                  className="command-palette-item"
                >
                  <KeyOutlined />
                  <span>Reset password</span>
                </Command.Item>
                <Command.Item
                  value="Unlock account"
                  onSelect={() => runAction(() => navigate('/users?filter=locked'))}
                  className="command-palette-item"
                >
                  <UnlockOutlined />
                  <span>Unlock account</span>
                </Command.Item>
                <Command.Item
                  value="Create DNS record"
                  onSelect={() => runAction(() => navigate('/dns?action=create'))}
                  className="command-palette-item"
                >
                  <GlobalOutlined />
                  <span>Create DNS record</span>
                </Command.Item>
                <Command.Item
                  value="Force replication sync"
                  onSelect={() => runAction(() => navigate('/replication'))}
                  className="command-palette-item"
                >
                  <NodeIndexOutlined />
                  <span>Force replication sync</span>
                </Command.Item>
              </Command.Group>
            )}

            {/* Navigation — filtered by role */}
            <Command.Group heading="Navigation" className="command-palette-group">
              <Command.Item value="Go to Dashboard" onSelect={() => runAction(() => navigate('/'))} className="command-palette-item">
                <DashboardOutlined /><span>Dashboard</span><kbd className="command-palette-shortcut">G D</kbd>
              </Command.Item>
              <Command.Item value="Go to Users" onSelect={() => runAction(() => navigate('/users'))} className="command-palette-item">
                <UserOutlined /><span>Users</span><kbd className="command-palette-shortcut">G U</kbd>
              </Command.Item>
              <Command.Item value="Go to Groups" onSelect={() => runAction(() => navigate('/groups'))} className="command-palette-item">
                <TeamOutlined /><span>Groups</span><kbd className="command-palette-shortcut">G G</kbd>
              </Command.Item>
              {isAdmin && (
                <Command.Item value="Go to Computers" onSelect={() => runAction(() => navigate('/computers'))} className="command-palette-item">
                  <DesktopOutlined /><span>Computers</span><kbd className="command-palette-shortcut">G C</kbd>
                </Command.Item>
              )}
              {isAdmin && (
                <Command.Item value="Go to Organizational Units OUs" onSelect={() => runAction(() => navigate('/ous'))} className="command-palette-item">
                  <ApartmentOutlined /><span>Organizational Units</span><kbd className="command-palette-shortcut">G O</kbd>
                </Command.Item>
              )}
              <Command.Item value="Go to DNS" onSelect={() => runAction(() => navigate('/dns'))} className="command-palette-item">
                <GlobalOutlined /><span>DNS</span><kbd className="command-palette-shortcut">G N</kbd>
              </Command.Item>
              {isAdmin && (
                <>
                  <Command.Item value="Go to Sites Services" onSelect={() => runAction(() => navigate('/sites'))} className="command-palette-item">
                    <CloudServerOutlined /><span>Sites & Services</span>
                  </Command.Item>
                  <Command.Item value="Go to Replication" onSelect={() => runAction(() => navigate('/replication'))} className="command-palette-item">
                    <NodeIndexOutlined /><span>Replication</span><kbd className="command-palette-shortcut">G R</kbd>
                  </Command.Item>
                  <Command.Item value="Go to Group Policy GPO" onSelect={() => runAction(() => navigate('/gpo'))} className="command-palette-item">
                    <SafetyCertificateOutlined /><span>Group Policy</span>
                  </Command.Item>
                  <Command.Item value="Go to Kerberos" onSelect={() => runAction(() => navigate('/kerberos'))} className="command-palette-item">
                    <KeyOutlined /><span>Kerberos</span>
                  </Command.Item>
                  <Command.Item value="Go to FSMO Roles" onSelect={() => runAction(() => navigate('/fsmo'))} className="command-palette-item">
                    <CrownOutlined /><span>FSMO Roles</span>
                  </Command.Item>
                  <Command.Item value="Go to Schema" onSelect={() => runAction(() => navigate('/schema'))} className="command-palette-item">
                    <DatabaseOutlined /><span>Schema</span>
                  </Command.Item>
                  <Command.Item value="Go to Audit Log" onSelect={() => runAction(() => navigate('/audit'))} className="command-palette-item">
                    <AuditOutlined /><span>Audit Log</span>
                  </Command.Item>
                  <Command.Item value="Go to Settings" onSelect={() => runAction(() => navigate('/settings'))} className="command-palette-item">
                    <SettingOutlined /><span>Settings</span>
                  </Command.Item>
                </>
              )}
            </Command.Group>
          </Command.List>
        </Command>
      </div>
    </div>
  );
}
