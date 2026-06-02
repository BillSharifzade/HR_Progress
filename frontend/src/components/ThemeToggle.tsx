import { useState } from 'react';
import { theme as antdTheme } from 'antd';
import { useThemeMode } from '../theme/ThemeContext';

const SunIcon = ({ size = 18 }: { size?: number }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <circle cx="12" cy="12" r="4.2" fill="currentColor" />
    <line x1="12" y1="2" x2="12" y2="4.5" />
    <line x1="12" y1="19.5" x2="12" y2="22" />
    <line x1="2" y1="12" x2="4.5" y2="12" />
    <line x1="19.5" y1="12" x2="22" y2="12" />
    <line x1="4.6" y1="4.6" x2="6.4" y2="6.4" />
    <line x1="17.6" y1="17.6" x2="19.4" y2="19.4" />
    <line x1="4.6" y1="19.4" x2="6.4" y2="17.6" />
    <line x1="17.6" y1="6.4" x2="19.4" y2="4.6" />
  </svg>
);

const MoonIcon = ({ size = 18 }: { size?: number }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor">
    <path d="M21 12.79A9 9 0 1 1 11.21 3a7 7 0 0 0 9.79 9.79z" />
  </svg>
);

const SUN_COLOR = '#F5A623';
const MOON_COLOR = '#A5B4FC';

export function ThemeToggle() {
  const { mode, toggle } = useThemeMode();
  const { token } = antdTheme.useToken();
  const isDark = mode === 'dark';

  const [hovered, setHovered] = useState(false);
  const [pressed, setPressed] = useState(false);

  const scale = pressed ? 0.88 : hovered ? 1.08 : 1;

  return (
    <button
      type="button"
      onClick={toggle}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => { setHovered(false); setPressed(false); }}
      onMouseDown={() => setPressed(true)}
      onMouseUp={() => setPressed(false)}
      onBlur={() => setPressed(false)}
      aria-label={isDark ? 'Включить светлую тему' : 'Включить тёмную тему'}
      title={isDark ? 'Светлая тема' : 'Тёмная тема'}
      style={{
        width: 34,
        height: 34,
        borderRadius: '50%',
        border: 'none',
        background: hovered ? token.colorFillSecondary : 'transparent',
        cursor: 'pointer',
        padding: 0,
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        position: 'relative',
        transform: `scale(${scale})`,
        transition: 'background 180ms ease, transform 220ms cubic-bezier(0.4, 0, 0.2, 1)',
        outline: 'none',
        WebkitTapHighlightColor: 'transparent',
      }}
    >
      <span style={{ position: 'relative', width: 18, height: 18, display: 'inline-block' }}>
        {/* Sun (visible in light mode) */}
        <span
          aria-hidden
          style={{
            position: 'absolute',
            inset: 0,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: SUN_COLOR,
            opacity: isDark ? 0 : 1,
            transform: isDark ? 'rotate(-90deg) scale(0.4)' : 'rotate(0deg) scale(1)',
            transition: 'opacity 320ms ease, transform 380ms cubic-bezier(0.4, 0, 0.2, 1)',
          }}
        >
          <SunIcon />
        </span>
        {/* Moon (visible in dark mode) */}
        <span
          aria-hidden
          style={{
            position: 'absolute',
            inset: 0,
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: MOON_COLOR,
            opacity: isDark ? 1 : 0,
            transform: isDark ? 'rotate(0deg) scale(1)' : 'rotate(90deg) scale(0.4)',
            transition: 'opacity 320ms ease, transform 380ms cubic-bezier(0.4, 0, 0.2, 1)',
          }}
        >
          <MoonIcon />
        </span>
      </span>
    </button>
  );
}
