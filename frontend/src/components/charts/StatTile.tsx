import { useEffect, useState, type ReactNode } from 'react';
import { Card, Typography } from 'antd';
import { prefersReducedMotion, useChartTheme } from './theme';

const { Text } = Typography;

/**
 * A headline number. Proportional figures (never tabular — equal-width digits
 * make a large standalone number look loose) in the app's own sans.
 */
export function StatTile({ title, value, suffix, hint, accent, delay = 0 }: {
  title: string;
  value: number | string | null;
  suffix?: ReactNode;
  hint?: ReactNode;
  /** Paint the figure in the series hue — reserve it for the lead tile. */
  accent?: boolean;
  delay?: number;
}) {
  const theme = useChartTheme();
  const display = useCountUp(typeof value === 'number' ? value : null, delay);

  return (
    <Card size="small" style={{ height: '100%' }}>
      <Text type="secondary" style={{ fontSize: 12 }}>{title}</Text>
      <div
        style={{
          marginTop: 4,
          fontSize: 30,
          lineHeight: 1.15,
          fontWeight: 650,
          color: accent ? theme.series : theme.ink,
          letterSpacing: '-0.02em',
        }}
      >
        {value === null ? '—' : typeof value === 'number' ? display : value}
        {suffix && <span style={{ fontSize: 15, fontWeight: 500, color: theme.inkMuted, marginLeft: 4 }}>{suffix}</span>}
      </div>
      {hint && <div style={{ marginTop: 2, fontSize: 12, color: theme.inkMuted }}>{hint}</div>}
    </Card>
  );
}

/** Counts a figure up on mount; returns the target immediately when the
 *  reader has asked for reduced motion. */
function useCountUp(target: number | null, delay: number): string {
  const [n, setN] = useState(() => (prefersReducedMotion() ? target ?? 0 : 0));

  useEffect(() => {
    if (target === null) return;
    if (prefersReducedMotion()) { setN(target); return; }
    const isFloat = !Number.isInteger(target);
    const duration = 900;
    let raf = 0;
    let start = 0;
    const tick = (t: number) => {
      if (!start) start = t;
      const p = Math.min(1, (t - start - delay) / duration);
      if (p < 0) { raf = requestAnimationFrame(tick); return; }
      // easeOutCubic — fast then settling, so the number lands rather than stops
      const eased = 1 - Math.pow(1 - p, 3);
      setN(isFloat ? Number((target * eased).toFixed(2)) : Math.round(target * eased));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, delay]);

  if (target === null) return '—';
  return Number.isInteger(target) ? String(n) : n.toFixed(2);
}
