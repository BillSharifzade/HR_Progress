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
