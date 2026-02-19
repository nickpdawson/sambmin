import { useState, useEffect } from 'react';
import { Layout, Menu, Breadcrumb, Typography, Button, Tooltip } from 'antd';
import {
  DashboardOutlined,
  UserOutlined,
  TeamOutlined,
  DesktopOutlined,
  ContactsOutlined,
  ApartmentOutlined,
  CloudServerOutlined,
  GlobalOutlined,
  NodeIndexOutlined,
  SafetyCertificateOutlined,
  KeyOutlined,
  CrownOutlined,
  DatabaseOutlined,
  AuditOutlined,
  SettingOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SunOutlined,
  MoonOutlined,
  SearchOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import type { MenuProps } from 'antd';
import CommandPalette from '../components/CommandPalette/CommandPalette';
import { useAuth } from '../hooks/useAuth';

const { Sider, Header, Content } = Layout;
const { Text } = Typography;

interface AppLayoutProps {
  isDark: boolean;
  onToggleTheme: () => void;
}

type MenuItem = Required<MenuProps>['items'][number];

const menuItems: MenuItem[] = [
  {
    key: 'overview',
    type: 'group',
    label: 'OVERVIEW',
    children: [
      { key: '/', icon: <DashboardOutlined />, label: 'Dashboard' },
    ],
  },
  {
    key: 'directory',
    type: 'group',
    label: 'DIRECTORY',
    children: [
      { key: '/users', icon: <UserOutlined />, label: 'Users' },
      { key: '/groups', icon: <TeamOutlined />, label: 'Groups' },
      { key: '/computers', icon: <DesktopOutlined />, label: 'Computers' },
      { key: '/contacts', icon: <ContactsOutlined />, label: 'Contacts' },
      { key: '/ous', icon: <ApartmentOutlined />, label: 'Organizational Units' },
    ],
  },
  {
    key: 'infrastructure',
    type: 'group',
    label: 'INFRASTRUCTURE',
    children: [
      { key: '/dns', icon: <GlobalOutlined />, label: 'DNS' },
      { key: '/sites', icon: <CloudServerOutlined />, label: 'Sites & Services' },
      { key: '/replication', icon: <NodeIndexOutlined />, label: 'Replication' },
    ],
  },
  {
    key: 'policy',
    type: 'group',
    label: 'POLICY & SECURITY',
    children: [
      { key: '/gpo', icon: <SafetyCertificateOutlined />, label: 'Group Policy' },
      { key: '/kerberos', icon: <KeyOutlined />, label: 'Kerberos' },
      { key: '/fsmo', icon: <CrownOutlined />, label: 'FSMO Roles' },
      { key: '/schema', icon: <DatabaseOutlined />, label: 'Schema' },
    ],
  },
  {
    key: 'system',
    type: 'group',
    label: 'SYSTEM',
    children: [
      { key: '/audit', icon: <AuditOutlined />, label: 'Audit Log' },
      { key: '/settings', icon: <SettingOutlined />, label: 'Settings' },
    ],
  },
];

const pathLabels: Record<string, string> = {
  '/': 'Dashboard',
  '/users': 'Users',
  '/groups': 'Groups',
  '/computers': 'Computers',
  '/contacts': 'Contacts',
  '/ous': 'Organizational Units',
  '/dns': 'DNS',
  '/sites': 'Sites & Services',
  '/replication': 'Replication',
  '/gpo': 'Group Policy',
  '/kerberos': 'Kerberos',
  '/fsmo': 'FSMO Roles',
  '/schema': 'Schema',
  '/audit': 'Audit Log',
  '/settings': 'Settings',
};

export default function AppLayout({ isDark, onToggleTheme }: AppLayoutProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();

  // Cmd+K / Ctrl+K keyboard shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCommandPaletteOpen((prev) => !prev);
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  const breadcrumbItems = [
    { title: 'Sambmin' },
    { title: pathLabels[location.pathname] || 'Unknown' },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        trigger={null}
        width={240}
        collapsedWidth={64}
        style={{
          borderRight: '1px solid var(--ant-color-border)',
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 100,
        }}
      >
        {/* Logo */}
        <div
          style={{
            height: 48,
            display: 'flex',
            alignItems: 'center',
            justifyContent: collapsed ? 'center' : 'flex-start',
            padding: collapsed ? 0 : '0 16px',
            borderBottom: '1px solid var(--ant-color-border)',
          }}
        >
          <Text strong style={{ fontSize: collapsed ? 14 : 18 }}>
            {collapsed ? 'S' : 'sambmin'}
          </Text>
        </div>

        {/* Navigation */}
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ border: 'none', marginTop: 8 }}
        />

        {/* Footer actions */}
        <div
          style={{
            position: 'absolute',
            bottom: 0,
            width: '100%',
            padding: '8px',
            borderTop: '1px solid var(--ant-color-border)',
            display: 'flex',
            justifyContent: collapsed ? 'center' : 'space-between',
            alignItems: 'center',
          }}
        >
          <Tooltip title={isDark ? 'Light mode' : 'Dark mode'}>
            <Button
              type="text"
              icon={isDark ? <SunOutlined /> : <MoonOutlined />}
              onClick={onToggleTheme}
              size="small"
            />
          </Tooltip>
          {!collapsed && (
            <Button
              type="text"
              icon={<MenuFoldOutlined />}
              onClick={() => setCollapsed(true)}
              size="small"
            />
          )}
        </div>
      </Sider>

      <Layout style={{ marginLeft: collapsed ? 64 : 240, transition: 'margin-left 0.2s' }}>
        {/* Top bar */}
        <Header
          style={{
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            height: 48,
            borderBottom: '1px solid var(--ant-color-border)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            {collapsed && (
              <Button
                type="text"
                icon={<MenuUnfoldOutlined />}
                onClick={() => setCollapsed(false)}
                size="small"
              />
            )}
            <Breadcrumb items={breadcrumbItems} />
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Tooltip title="Search (⌘K)">
              <Button
                type="text"
                icon={<SearchOutlined />}
                size="small"
                onClick={() => setCommandPaletteOpen(true)}
              />
            </Tooltip>
            {user && (
              <>
                <Text type="secondary" style={{ fontSize: 13 }}>{user.username}</Text>
                <Tooltip title="Sign out">
                  <Button
                    type="text"
                    icon={<LogoutOutlined />}
                    size="small"
                    onClick={async () => { await logout(); navigate('/login'); }}
                  />
                </Tooltip>
              </>
            )}
          </div>
        </Header>

        {/* Main content */}
        <Content style={{ padding: 24 }}>
          <Outlet />
        </Content>
      </Layout>

      {/* Command Palette */}
      <CommandPalette
        open={commandPaletteOpen}
        onOpenChange={setCommandPaletteOpen}
      />
    </Layout>
  );
}
