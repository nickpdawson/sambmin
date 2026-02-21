import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { api, ApiError } from '../api/client';

interface User {
  username: string;
  dn: string;
  groups: string[];
  expires?: string;
}

// Groups that grant admin access to the full management UI
const ADMIN_GROUPS = ['DOMAIN ADMINS', 'ENTERPRISE ADMINS', 'SCHEMA ADMINS', 'ACCOUNT OPERATORS'];

function checkIsAdmin(groups: string[] | undefined): boolean {
  if (!groups || groups.length === 0) return false;
  return groups.some((g) => {
    const upper = g.toUpperCase();
    // Match full DNs like "CN=Domain Admins,CN=Users,DC=..." or plain names like "Domain Admins"
    return ADMIN_GROUPS.some((ag) => upper.includes(`CN=${ag},`) || upper === ag);
  });
}

interface AuthContextType {
  user: User | null;
  isAdmin: boolean;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Check for existing session on mount
  useEffect(() => {
    api.get<User>('/auth/me')
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const result = await api.post<User>('/auth/login', { username, password });
    setUser(result);
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.post('/auth/logout');
    } catch {
      // Logout even if API fails
    }
    setUser(null);
  }, []);

  const isAdmin = checkIsAdmin(user?.groups);

  return (
    <AuthContext.Provider value={{ user, isAdmin, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

export { ApiError };
