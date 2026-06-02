import { Form, Input, Button, Typography, App, theme as antdTheme } from 'antd';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import { useAuth } from '../../auth/useAuth';
import { useThemeMode } from '../../theme/ThemeContext';
import { BRAND_PRIMARY, BRAND_PRIMARY_DEEP, SPLASH_BG_DARK, SPLASH_BG_LIGHT } from '../../theme';
import { ThemeToggle } from '../../components/ThemeToggle';

interface FormValues {
  username: string;
  password: string;
}

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation() as { state?: { from?: { pathname?: string } } };
  const [submitting, setSubmitting] = useState(false);
  const { message } = App.useApp();
  const { mode } = useThemeMode();
  const { token } = antdTheme.useToken();
  const isDark = mode === 'dark';

  const handleSubmit = async (v: FormValues) => {
    setSubmitting(true);
    try {
      await login(v.username, v.password);
      const dest = location.state?.from?.pathname || '/';
      navigate(dest, { replace: true });
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: { message?: string } } } };
      message.error(e?.response?.data?.error?.message ?? 'Ошибка входа');
    } finally {
      setSubmitting(false);
    }
  };

  const decorTint = isDark ? 'rgba(99,102,241,0.10)' : 'rgba(79,70,229,0.06)';
  const subText = isDark ? 'rgba(255,255,255,0.55)' : '#6b7280';
  const labelColor = isDark ? 'rgba(255,255,255,0.85)' : '#374151';
  const headingColor = isDark ? '#fff' : '#0d1b3e';

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      background: isDark ? SPLASH_BG_DARK : SPLASH_BG_LIGHT,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      fontFamily: '"Inter", -apple-system, sans-serif',
      overflow: 'hidden',
      transition: 'background 280ms ease',
    }}>
      {/* Top-right theme toggle */}
      <div style={{ position: 'absolute', top: 24, right: 24, zIndex: 2 }}>
        <ThemeToggle />
      </div>

      {/* Decorative circles */}
      <div style={{
        position: 'absolute', top: -120, right: -120,
        width: 480, height: 480, borderRadius: '50%',
        background: decorTint,
        pointerEvents: 'none',
      }} />
      <div style={{
        position: 'absolute', bottom: -80, left: -80,
        width: 360, height: 360, borderRadius: '50%',
        background: decorTint,
        pointerEvents: 'none',
      }} />
      <div style={{
        position: 'absolute', top: '40%', left: '15%',
        width: 200, height: 200, borderRadius: '50%',
        background: decorTint,
        pointerEvents: 'none',
      }} />

      {/* Card */}
      <div style={{
        position: 'relative', zIndex: 1,
        width: '100%', maxWidth: 420,
        margin: '0 24px',
        background: token.colorBgContainer,
        borderRadius: 16,
        padding: '40px 40px 36px',
        boxShadow: isDark
          ? '0 8px 40px rgba(0,0,0,0.5)'
          : '0 8px 40px rgba(79,70,229,0.12)',
        transition: 'background 280ms ease, box-shadow 280ms ease',
      }}>
        {/* Brand */}
        <div style={{ marginBottom: 32, textAlign: 'center' }}>
          <Typography.Title level={2} style={{ margin: 0, fontWeight: 700, color: BRAND_PRIMARY, letterSpacing: '-0.5px' }}>
            HR Progress
          </Typography.Title>
          <Typography.Text style={{ color: subText, fontSize: 14, marginTop: 4, display: 'block' }}>
            Платформа развития сотрудников
          </Typography.Text>
        </div>

        <div style={{ marginBottom: 28 }}>
          <Typography.Title level={4} style={{ margin: 0, fontWeight: 700, color: headingColor, letterSpacing: '-0.3px' }}>
            Добро пожаловать
          </Typography.Title>
          <Typography.Text style={{ color: subText, fontSize: 13, marginTop: 4, display: 'block' }}>
            Войдите в свою учётную запись
          </Typography.Text>
        </div>

        <Form<FormValues> layout="vertical" onFinish={handleSubmit} disabled={submitting}>
          <Form.Item
            name="username"
            label={<span style={{ fontWeight: 500, color: labelColor, fontSize: 13 }}>Имя пользователя</span>}
            rules={[{ required: true, message: 'Введите имя пользователя' }]}
          >
            <Input
              prefix={<UserOutlined style={{ color: subText }} />}
              size="large"
              autoFocus
              autoComplete="username"
              style={{ borderRadius: 10 }}
            />
          </Form.Item>

          <Form.Item
            name="password"
            label={<span style={{ fontWeight: 500, color: labelColor, fontSize: 13 }}>Пароль</span>}
            rules={[{ required: true, message: 'Введите пароль' }]}
            style={{ marginBottom: 24 }}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: subText }} />}
              size="large"
              autoComplete="current-password"
              style={{ borderRadius: 10 }}
            />
          </Form.Item>

          <Button
            type="primary"
            htmlType="submit"
            block
            loading={submitting}
            size="large"
            style={{
              borderRadius: 10,
              height: 46,
              fontWeight: 600,
              background: `linear-gradient(90deg, ${BRAND_PRIMARY} 0%, ${BRAND_PRIMARY_DEEP} 100%)`,
              border: 'none',
            }}
          >
            Войти
          </Button>
        </Form>
      </div>
    </div>
  );
}
