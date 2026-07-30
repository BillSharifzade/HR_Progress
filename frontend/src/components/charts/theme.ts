import { useThemeMode } from '../../theme/ThemeContext';

/**
 * Chart palette, anchored on the app's brand indigo (#4F46E5).
 *
 * Every value below was checked with the data-viz validator against the
 * surfaces these charts actually render on — white in light mode, the AntD
 * container `#1A1B2E` in dark. The dark column is its own set of steps chosen
 * for the dark band, not an automatic lightening of the light one:
 *
 *   light diverging  #4F46E5 / #DC2626  → band, chroma, CVD ΔE 29.8, contrast — PASS
 *   dark  diverging  #6366F1 / #EF4444  → band, chroma, CVD ΔE 27.0, contrast — PASS
 *   light ordinal    #9BABFB…#3730A3    → monotone L, ΔL ≥ .06, light end 2.18:1 — PASS
 *   dark  ordinal    #C7D2FE…#4F46E5    → monotone L, ΔL ≥ .06, light end 2.69:1 — PASS
 *
 * If you change a hex, re-run the validator before shipping it.
 */
export interface ChartTheme {
  surface: string;
  /** Single hue for magnitude — one series, one color, every bar. */
  series: string;
  /** Ordered scale for genuinely ordinal categories (score bands). */
  ordinal: string[];
  /** Diverging poles: above/below a target. Midpoint is `grid`. */
  positive: string;
  negative: string;
  grid: string;
  axis: string;
  ink: string;
  inkSecondary: string;
  inkMuted: string;
}

const LIGHT: ChartTheme = {
  surface: '#FFFFFF',
  series: '#4F46E5',
  ordinal: ['#9BABFB', '#818CF8', '#6366F1', '#4F46E5', '#3730A3'],
  positive: '#4F46E5',
  negative: '#DC2626',
  grid: '#EDEDF2',
  axis: '#D9D9E3',
  ink: '#0B0B12',
  inkSecondary: '#52525E',
  inkMuted: '#8C8C9A',
};

const DARK: ChartTheme = {
  surface: '#1A1B2E',
  series: '#6366F1',
  ordinal: ['#C7D2FE', '#A5B4FC', '#818CF8', '#6366F1', '#4F46E5'],
  positive: '#6366F1',
  negative: '#EF4444',
  grid: '#2C2E48',
  axis: '#3A3C5A',
  ink: '#FFFFFF',
  inkSecondary: '#C6C7D8',
  inkMuted: '#9092AC',
};

/** Status tokens are fixed in both modes and always ship with a label. */
export const STATUS = {
  good: '#0CA30C',
  warning: '#FAB219',
  critical: '#D03B3B',
} as const;

export function useChartTheme(): ChartTheme {
  const { mode } = useThemeMode();
  return mode === 'dark' ? DARK : LIGHT;
}

/** Honour the OS reduced-motion setting — chart entry animations are opt-out. */
export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * "Nice" axis maximum so gridlines land on round numbers. The rungs are close
 * together on purpose: a coarse ladder (1/2/5/10) sends a top value of 22 to a
 * max of 50 and leaves half the plot empty.
 */
export function niceMax(value: number): number {
  if (value <= 0) return 1;
  const mag = Math.pow(10, Math.floor(Math.log10(value)));
  const norm = value / mag;
  const rungs = [1, 1.2, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10];
  const step = rungs.find(r => norm <= r + 1e-9) ?? 10;
  return step * mag;
}

/**
 * Tick values including 0 and max. Segment count is chosen so the ticks land
 * on whole numbers — these axes count people and marks, so "0, 0.25, 0.5" is
 * never the right ladder.
 */
export function ticksFor(max: number): number[] {
  const usable = [4, 5, 3, 2].find(c => Number.isInteger(max / c)) ?? Math.max(1, Math.min(Math.round(max), 4));
  const out: number[] = [];
  for (let i = 0; i <= usable; i++) out.push((max / usable) * i);
  return out;
}
