import { ConfigProvider, theme, Spin, App as AntApp } from 'antd';
import enUS from 'antd/locale/en_US';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { lightTheme, darkTheme } from './theme/tokens';
import { useTheme } from './hooks/useTheme';
import { AuthProvider, useAuth } from './hooks/useAuth';
import AppLayout from './layouts/AppLayout';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import Users from './pages/Users';
import Groups from './pages/Groups';
import Computers from './pages/Computers';
import OUs from './pages/OUs';
import DNS from './pages/DNS';
import Sites from './pages/Sites';
import Replication from './pages/Replication';
import GPO from './pages/GPO';
import Kerberos from './pages/Kerberos';
import FSMO from './pages/FSMO';
import Schema from './pages/Schema';
import AuditLog from './pages/AuditLog';
import Settings from './pages/Settings';
import type { ReactNode } from 'react';

function RequireAuth({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}

export default function App() {
  const { toggle, isDark } = useTheme();
  const themeConfig = isDark ? darkTheme : lightTheme;

  return (
    <ConfigProvider
      locale={enUS}
      theme={{
        ...themeConfig,
        algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
      }}
    >
      <AntApp>
        <BrowserRouter>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route element={
                <RequireAuth>
                  <AppLayout isDark={isDark} onToggleTheme={toggle} />
                </RequireAuth>
              }>
                <Route path="/" element={<Dashboard />} />
                <Route path="/users" element={<Users />} />
                <Route path="/groups" element={<Groups />} />
                <Route path="/computers" element={<Computers />} />
                <Route path="/ous" element={<OUs />} />
                <Route path="/dns" element={<DNS />} />
                <Route path="/sites" element={<Sites />} />
                <Route path="/replication" element={<Replication />} />
                <Route path="/gpo" element={<GPO />} />
                <Route path="/kerberos" element={<Kerberos />} />
                <Route path="/fsmo" element={<FSMO />} />
                <Route path="/schema" element={<Schema />} />
                <Route path="/audit" element={<AuditLog />} />
                <Route path="/settings" element={<Settings />} />
              </Route>
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </AntApp>
    </ConfigProvider>
  );
}
