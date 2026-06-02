import { useState, type ReactNode } from 'react';
import { Layout, Menu, Avatar, Badge, Dropdown, Typography, theme, Tooltip } from 'antd';
import { useQuery } from '@tanstack/react-query';
import {
  DashboardOutlined,
  TeamOutlined,
  AppstoreOutlined,
  CalendarOutlined,
  RiseOutlined,
  BarChartOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import { Link, useLocation } from 'react-router-dom';

import { useAuth } from '../auth/useAuth';
import { useThemeMode } from '../theme/ThemeContext';
import { ThemeToggle } from './ThemeToggle';
import { ActivePeriodNotifier } from './ActivePeriodBanner';
import { listMyAssessmentPeriods } from '../api/competency';

const { Sider, Content } = Layout;

export function AppShell({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(false);

  const { token } = theme.useToken();
  const { mode } = useThemeMode();
  const isDark = mode === 'dark';
  const isAdmin = user?.roles.includes('HR_ADMIN') ?? false;

  const { data: myPeriods = [] } = useQuery({
    queryKey: ['my-assessment-periods'],
    queryFn: listMyAssessmentPeriods,
    enabled: !!user,
  });
  const activePeriodCount = myPeriods.filter(p => p.is_active).length;

  const assessmentsLabel = (
    <Link to="/assessments">
      <Badge
        count={activePeriodCount}
        size="small"
        offset={[10, 0]}
        styles={{ indicator: { boxShadow: 'none' } }}
      >
        <span>Мои оценки</span>
      </Badge>
    </Link>
  );

  const items = isAdmin
    ? [
        { key: '/', icon: <DashboardOutlined />, label: <Link to="/">Дашборд</Link> },
        { key: '/workers', icon: <TeamOutlined />, label: <Link to="/workers">Сотрудники</Link> },
        { key: '/competencies', icon: <AppstoreOutlined />, label: <Link to="/competencies">Компетенции</Link> },
        { key: '/assessments', icon: <RiseOutlined />, label: assessmentsLabel },
        { key: '/development', icon: <RiseOutlined />, label: 'Развитие', disabled: true },
        { key: '/calendar', icon: <CalendarOutlined />, label: 'Календарь', disabled: true },
        { key: '/reports', icon: <BarChartOutlined />, label: 'Отчёты', disabled: true },
        { key: '/admin', icon: <SettingOutlined />, label: <Link to="/admin">Администрирование</Link> },
      ]
    : [
        { key: '/workers', icon: <TeamOutlined />, label: <Link to="/workers">Сотрудники</Link> },
        { key: '/competencies', icon: <AppstoreOutlined />, label: <Link to="/competencies">Компетенции</Link> },
        { key: '/assessments', icon: <RiseOutlined />, label: assessmentsLabel },
        { key: '/development', icon: <RiseOutlined />, label: 'Развитие', disabled: true },
      ];

  const selectedKey = items.find((item) => {
    if (item.key === '/') return location.pathname === '/';
    return location.pathname.startsWith(item.key);
  })?.key ?? '/';

  const userMenuItems = [
    {
      key: 'logout',
      label: 'Выйти',
      icon: <LogoutOutlined />,
      onClick: () => logout(),
      danger: true,
    },
  ];

  const displayName = user?.full_name ?? user?.username ?? '';

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={240}
        theme={isDark ? 'dark' : 'light'}
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          height: '100vh',
          zIndex: 100,
          borderRight: `1px solid ${token.colorBorderSecondary}`,
          willChange: 'width',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          background: token.colorBgContainer,
          transition: 'background 250ms ease',
        }}
      >
        {/* Inner flex column to stretch full height */}
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          {/* Logo + theme toggle + collapse */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              flexDirection: collapsed ? 'column' : 'row',
              gap: collapsed ? 4 : 0,
              justifyContent: collapsed ? 'center' : 'space-between',
              padding: collapsed ? '12px 0' : '12px 12px 12px 20px',
              transition: 'padding 0.2s',
              flexShrink: 0,
            }}
          >
            {!collapsed && (
              <Typography.Text
                strong
                style={{ fontSize: 17, color: token.colorPrimary, whiteSpace: 'nowrap' }}
              >
                HR Progress
              </Typography.Text>
            )}
            <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}>
              <ThemeToggle />
              <div
                onClick={() => setCollapsed((c) => !c)}
                style={{
                  cursor: 'pointer',
                  color: token.colorTextSecondary,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: 34,
                  height: 34,
                  borderRadius: '50%',
                  transition: 'background 0.15s',
                  flexShrink: 0,
                }}
                onMouseEnter={e => (e.currentTarget.style.background = token.colorFillSecondary)}
                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
              >
                {collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              </div>
            </div>
          </div>

          {/* Nav menu — takes up remaining space */}
          <Menu
            mode="inline"
            inlineCollapsed={collapsed}
            selectedKeys={[selectedKey]}
            items={items}
            style={{ borderRight: 0, flex: 1, overflowY: 'auto', overflowX: 'hidden' }}
          />

          {/* User profile at bottom */}
          <Dropdown menu={{ items: userMenuItems }} placement="topLeft" trigger={['click']}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: collapsed ? '14px 0' : '14px 16px',
                justifyContent: collapsed ? 'center' : 'flex-start',
                cursor: 'pointer',
                borderTop: `1px solid ${token.colorBorderSecondary}`,
                transition: 'background 0.15s, padding 0.2s',
                flexShrink: 0,
              }}
              onMouseEnter={e => (e.currentTarget.style.background = token.colorFillAlter)}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              {collapsed ? (
                <Tooltip title={displayName} placement="right">
                  <Avatar size={32} icon={<UserOutlined />} style={{ flexShrink: 0 }} />
                </Tooltip>
              ) : (
                <>
                  <Avatar size={32} icon={<UserOutlined />} style={{ flexShrink: 0 }} />
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <Typography.Text
                      strong
                      ellipsis
                      style={{ display: 'block', fontSize: 13, lineHeight: 1.3 }}
                    >
                      {displayName}
                    </Typography.Text>
                    {user?.position && (
                      <Typography.Text
                        type="secondary"
                        ellipsis
                        style={{ display: 'block', fontSize: 11 }}
                      >
                        {user.position}
                      </Typography.Text>
                    )}
                  </div>
                </>
              )}
            </div>
          </Dropdown>
        </div>
      </Sider>

      <Layout style={{ marginLeft: collapsed ? 80 : 240, transition: 'margin-left 0.2s' }}>
        <Content
          style={{
            padding: 24,
            background: token.colorBgLayout,
            minHeight: '100vh',
            transition: 'background 250ms ease',
          }}
        >
          <ActivePeriodNotifier />
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}
