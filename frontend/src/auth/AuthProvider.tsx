import { createContext, useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import { Spin } from 'antd';
import { useNavigate } from 'react-router-dom';

import { apiLogin, apiLogout, apiMe, apiRefresh } from '../api/auth';
import { setAccessToken, setUnauthorizedHandler } from '../api/client';
import type { User } from '../types';
import { useThemeMode } from '../theme/ThemeContext';
import { BRAND_PRIMARY, SPLASH_BG_DARK, SPLASH_BG_LIGHT } from '../theme';

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
  reload: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const { mode } = useThemeMode();

  const reload = useCallback(async () => {
    try {
      const u = await apiMe();
      setUser(u);
    } catch {
      setUser(null);
    }
  }, []);

  // On mount: try refresh + me to restore session.
  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const r = await apiRefresh();
        if (!alive) return;
        setAccessToken(r.access_token);
        setUser(r.user);
      } catch {
        if (alive) setUser(null);
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  // Hook into 401-after-failed-refresh to bounce to /login.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      setUser(null);
      setAccessToken(null);
      navigate('/login', { replace: true });
    });
  }, [navigate]);

  const login = useCallback(async (username: string, password: string) => {
    const r = await apiLogin(username, password);
    setAccessToken(r.access_token);
    setUser(r.user);
    return r.user;
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiLogout();
    } catch {
      // ignore
    }
    setAccessToken(null);
    setUser(null);
    navigate('/login', { replace: true });
  }, [navigate]);

  const value = useMemo(() => ({ user, loading, login, logout, reload }), [user, loading, login, logout, reload]);

  if (loading) {
    const isDark = mode === 'dark';
    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          gap: 20,
          background: isDark ? SPLASH_BG_DARK : SPLASH_BG_LIGHT,
          transition: 'background 250ms ease',
        }}
      >
        <div
          style={{
            fontFamily: '"Inter", -apple-system, sans-serif',
            fontSize: 22,
            fontWeight: 700,
            color: BRAND_PRIMARY,
            letterSpacing: '-0.3px',
          }}
        >
          HR Progress
        </div>
        <Spin size="large" />
      </div>
    );
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
