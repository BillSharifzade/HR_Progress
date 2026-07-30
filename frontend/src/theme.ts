import type { ThemeConfig } from 'antd';

// Single source of truth for the brand color (deeper indigo).
export const BRAND_PRIMARY = '#4F46E5';
export const BRAND_PRIMARY_HOVER = '#6366F1';
export const BRAND_PRIMARY_DEEP = '#4338CA';

export const baseTokens: ThemeConfig['token'] = {
  colorPrimary: BRAND_PRIMARY,
  colorInfo: BRAND_PRIMARY,
  borderRadius: 8,
  fontFamily:
    '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
};

export const darkTokens: ThemeConfig['token'] = {
  ...baseTokens,
  colorBgLayout: '#13131F',
  colorBgContainer: '#1A1B2E',
  colorBgElevated: '#22243A',
  colorBorderSecondary: '#2C2E48',
};

// Used by login page / loading splash to pick a backdrop matching the mode.
export const SPLASH_BG_LIGHT = '#EEF0FD';
export const SPLASH_BG_DARK = '#0E0F1B';

/**
 * Key ("critical") competencies are marked in purple — red is reserved for the
 * >4 divergence flag. The dark step is lighter on purpose: the light-mode
 * purple measures only 2.44:1 on the dark container and this color is used for
 * text, not just an icon. Light 6.94:1 on white, dark 5.77:1 on #1A1B2E.
 */
export const CRITICAL_PURPLE_LIGHT = '#722ED1';
export const CRITICAL_PURPLE_DARK = '#B37FEB';

export function criticalPurple(isDark: boolean): string {
  return isDark ? CRITICAL_PURPLE_DARK : CRITICAL_PURPLE_LIGHT;
}
