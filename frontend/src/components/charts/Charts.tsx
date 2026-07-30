import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from 'react';
import { niceMax, prefersReducedMotion, ticksFor, useChartTheme } from './theme';

/* ────────────────────────────────────────────────────────────────────────────
 * Shared bits
 *
 * Every chart measures its container and lays out in real pixels. SVG geometry
 * attributes take neither `calc()` nor mixed units, and scaling a nested
 * viewBox with preserveAspectRatio="none" would distort corner radii and
 * stroke widths — so the arithmetic happens here instead.
 * ──────────────────────────────────────────────────────────────────────────── */

/** Bar/segment ends are rounded 4px and anchored flat to the baseline. */
const R = 4;
const EASE = 'cubic-bezier(.22,.8,.3,1)';

/** Width of the host element, tracked across resizes. */
function useMeasure(): [React.RefObject<HTMLDivElement>, number] {
  const ref = useRef<HTMLDivElement>(null);
  const [w, setW] = useState(0);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const apply = () => setW(el.clientWidth);
    apply();
    const ro = new ResizeObserver(apply);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  return [ref, w];
}

/**
 * Rounded-on-one-end bar path. `side` says which end gets the radius, so the
 * mark stays anchored to its baseline instead of floating as a pill.
 */
function barPath(x: number, y: number, w: number, h: number, side: 'top' | 'right' | 'left'): string {
  if (h <= 0 || w <= 0) return '';
  const r = Math.max(0, Math.min(R, side === 'top' ? w / 2 : h / 2, side === 'top' ? h : w));
  switch (side) {
    case 'top':
      return `M${x},${y + h} L${x},${y + r} Q${x},${y} ${x + r},${y} L${x + w - r},${y} Q${x + w},${y} ${x + w},${y + r} L${x + w},${y + h} Z`;
    case 'right':
      return `M${x},${y} L${x + w - r},${y} Q${x + w},${y} ${x + w},${y + r} L${x + w},${y + h - r} Q${x + w},${y + h} ${x + w - r},${y + h} L${x},${y + h} Z`;
    case 'left':
      return `M${x + w},${y} L${x + r},${y} Q${x},${y} ${x},${y + r} L${x},${y + h - r} Q${x},${y + h} ${x + r},${y + h} L${x + w},${y + h} Z`;
  }
}

/** Fires once after mount so marks can transition from their zero state. */
function useEntered(): boolean {
  const [entered, setEntered] = useState(prefersReducedMotion());
  useEffect(() => {
    if (entered) return;
    const raf = requestAnimationFrame(() => setEntered(true));
    return () => cancelAnimationFrame(raf);
  }, [entered]);
  return entered;
}

interface TipState { x: number; y: number; content: ReactNode }

function useTooltip(hostRef: React.RefObject<HTMLDivElement>) {
  const [tip, setTip] = useState<TipState | null>(null);
  const show = (e: React.MouseEvent, content: ReactNode) => {
    const host = hostRef.current;
    if (!host) return;
    const r = host.getBoundingClientRect();
    setTip({ x: e.clientX - r.left, y: e.clientY - r.top, content });
  };
  return { tip, show, hide: () => setTip(null) };
}

function TooltipLayer({ tip }: { tip: TipState | null }) {
  const theme = useChartTheme();
  if (!tip) return null;
  return (
    <div
      style={{
        position: 'absolute', left: tip.x, top: tip.y,
        transform: 'translate(-50%, -125%)', pointerEvents: 'none',
        background: theme.surface, color: theme.ink,
        border: `1px solid ${theme.axis}`, borderRadius: 8,
        padding: '6px 10px', fontSize: 12, lineHeight: 1.45, whiteSpace: 'nowrap',
        boxShadow: '0 6px 20px rgba(0,0,0,.18)', zIndex: 5,
      }}
    >
      {tip.content}
    </div>
  );
}

/**
 * Truncate to fit an approximate pixel width. The per-character estimate is
 * deliberately generous — Cyrillic runs wider than Latin in Inter, and a label
 * that overruns its lane gets clipped by the card edge.
 */
function clip(text: string, px: number, fontSize = 12): string {
  const perChar = fontSize * 0.62;
  const maxChars = Math.max(3, Math.floor(px / perChar));
  return text.length > maxChars ? `${text.slice(0, maxChars - 1)}…` : text;
}

/* ────────────────────────────────────────────────────────────────────────────
 * Horizontal bar — magnitude across categories
 * ──────────────────────────────────────────────────────────────────────────── */

export interface BarDatum { label: string; value: number }

export function BarChart({ data, unit = '', ordinal = false, labelWidth = 150 }: {
  data: BarDatum[];
  unit?: string;
  /** True only when the categories carry a real order (grades, tiers). */
  ordinal?: boolean;
  labelWidth?: number;
}) {
  const theme = useChartTheme();
  const entered = useEntered();
  const [hostRef, W] = useMeasure();
  const { tip, show, hide } = useTooltip(hostRef);

  const rowH = 30, gapY = 10, valueW = 40;
  const barH = rowH - gapY;
  const height = data.length * rowH + 6;
  const L = Math.min(labelWidth, Math.max(90, W * 0.42));
  const plotW = Math.max(0, W - L - valueW);
  const max = niceMax(Math.max(...data.map(d => d.value), 1));
  const total = data.reduce((s, d) => s + d.value, 0);
  const colorAt = (i: number) =>
    ordinal ? theme.ordinal[Math.round((i / Math.max(data.length - 1, 1)) * (theme.ordinal.length - 1))] : theme.series;

  return (
    <div ref={hostRef} style={{ position: 'relative', width: '100%' }}>
      <svg width="100%" height={height} role="img" style={{ display: 'block' }}>
        {W > 0 && (
          <>
            {ticksFor(max).map(t => (
              <line
                key={t} x1={L + (t / max) * plotW} x2={L + (t / max) * plotW}
                y1={0} y2={data.length * rowH}
                stroke={theme.grid} strokeWidth={1}
              />
            ))}
            {data.map((d, i) => {
              const w = entered ? (d.value / max) * plotW : 0;
              const y = i * rowH;
              const pct = total > 0 ? Math.round((d.value / total) * 100) : 0;
              return (
                <g key={d.label}>
                  <path
                    d={barPath(L, y + gapY / 2, Math.max(w, 0.01), barH, 'right')}
                    fill={colorAt(i)}
                    style={{ transition: prefersReducedMotion() ? undefined : `d ${520 + i * 45}ms ${EASE}` }}
                  />
                  <text
                    x={L - 10} y={y + rowH / 2} textAnchor="end" dominantBaseline="central"
                    fontSize={12} fill={theme.inkSecondary}
                  >
                    {clip(d.label, L - 14)}
                  </text>
                  <text
                    x={W} y={y + rowH / 2} textAnchor="end" dominantBaseline="central"
                    fontSize={12} fontWeight={600} fill={theme.ink}
                    style={{ fontVariantNumeric: 'tabular-nums' }}
                  >
                    {d.value}
                  </text>
                  <rect
                    x={0} y={y} width={W} height={rowH} fill="transparent"
                    onMouseMove={e => show(e, <><strong>{d.label}</strong><br />{d.value}{unit && ` ${unit}`} · {pct}%</>)}
                    onMouseLeave={hide}
                  />
                </g>
              );
            })}
          </>
        )}
      </svg>
      <TooltipLayer tip={tip} />
    </div>
  );
}

/* ────────────────────────────────────────────────────────────────────────────
 * Column chart — vertical; short category labels only
 * ──────────────────────────────────────────────────────────────────────────── */

export function ColumnChart({ data, ordinal = false, unit = '', height = 220 }: {
  data: BarDatum[];
  ordinal?: boolean;
  unit?: string;
  height?: number;
}) {
  const theme = useChartTheme();
  const entered = useEntered();
  const [hostRef, W] = useMeasure();
  const { tip, show, hide } = useTooltip(hostRef);

  const padL = 38, padB = 28, padT = 12;
  const plotH = height - padB - padT;
  const plotW = Math.max(0, W - padL);
  const max = niceMax(Math.max(...data.map(d => d.value), 1));
  const total = data.reduce((s, d) => s + d.value, 0);
  const slot = plotW / Math.max(data.length, 1);
  const barW = slot * 0.6;
  const colorAt = (i: number) =>
    ordinal ? theme.ordinal[Math.round((i / Math.max(data.length - 1, 1)) * (theme.ordinal.length - 1))] : theme.series;

  return (
    <div ref={hostRef} style={{ position: 'relative', width: '100%' }}>
      <svg width="100%" height={height} role="img" style={{ display: 'block' }}>
        {W > 0 && (
          <>
            {ticksFor(max).map(t => {
              const y = padT + plotH - (t / max) * plotH;
              return (
                <g key={t}>
                  <line x1={padL} x2={W} y1={y} y2={y} stroke={theme.grid} strokeWidth={1} />
                  <text x={padL - 8} y={y} textAnchor="end" dominantBaseline="central" fontSize={11} fill={theme.inkMuted} style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {Math.round(t)}
                  </text>
                </g>
              );
            })}
            <line x1={padL} x2={W} y1={padT + plotH} y2={padT + plotH} stroke={theme.axis} strokeWidth={1} />
            {data.map((d, i) => {
              const h = entered ? (d.value / max) * plotH : 0;
              const cx = padL + slot * (i + 0.5);
              const pct = total > 0 ? Math.round((d.value / total) * 100) : 0;
              return (
                <g key={d.label}>
                  <path
                    d={barPath(cx - barW / 2, padT + plotH - h, barW, Math.max(h, 0.01), 'top')}
                    fill={colorAt(i)}
                    style={{ transition: prefersReducedMotion() ? undefined : `d ${520 + i * 60}ms ${EASE}` }}
                  />
                  <text x={cx} y={padT + plotH + 16} textAnchor="middle" fontSize={11} fill={theme.inkMuted}>
                    {clip(d.label, slot - 4, 11)}
                  </text>
                  <rect
                    x={padL + slot * i} y={padT} width={slot} height={plotH} fill="transparent"
                    onMouseMove={e => show(e, <><strong>{d.label}</strong><br />{d.value}{unit && ` ${unit}`} · {pct}%</>)}
                    onMouseLeave={hide}
                  />
                </g>
              );
            })}
          </>
        )}
      </svg>
      <TooltipLayer tip={tip} />
    </div>
  );
}

/* ────────────────────────────────────────────────────────────────────────────
 * Diverging bar — distance above/below a target, centred on zero
 * ──────────────────────────────────────────────────────────────────────────── */

export interface DivergingDatum {
  label: string;
  value: number; // the delta, signed
  actual: number;
  target: number;
  meta?: string;
}

export function DivergingBar({ data, labelWidth = 176 }: {
  data: DivergingDatum[];
  labelWidth?: number;
}) {
  const theme = useChartTheme();
  const entered = useEntered();
  const [hostRef, W] = useMeasure();
  const { tip, show, hide } = useTooltip(hostRef);

  const rowH = 30, gapY = 10, valueW = 46;
  const barH = rowH - gapY;
  const height = data.length * rowH + 6;
  const L = Math.min(labelWidth, Math.max(96, W * 0.4));
  const plotW = Math.max(0, W - L - valueW);
  const mid = L + plotW / 2;
  const max = Math.max(...data.map(d => Math.abs(d.value)), 0.5);

  return (
    <div ref={hostRef} style={{ position: 'relative', width: '100%' }}>
      <svg width="100%" height={height} role="img" style={{ display: 'block' }}>
        {W > 0 && (
          <>
            {data.map((d, i) => {
              const half = (Math.abs(d.value) / max) * (plotW / 2);
              const w = entered ? half : 0;
              const neg = d.value < 0;
              const y = i * rowH;
              return (
                <g key={d.label}>
                  <path
                    d={barPath(neg ? mid - w : mid, y + gapY / 2, Math.max(w, 0.01), barH, neg ? 'left' : 'right')}
                    fill={neg ? theme.negative : theme.positive}
                    style={{ transition: prefersReducedMotion() ? undefined : `d ${520 + i * 45}ms ${EASE}` }}
                  />
                  <text x={L - 10} y={y + rowH / 2} textAnchor="end" dominantBaseline="central" fontSize={12} fill={theme.inkSecondary}>
                    {clip(d.label, L - 14)}
                  </text>
                  <text
                    x={W} y={y + rowH / 2} textAnchor="end" dominantBaseline="central"
                    fontSize={12} fontWeight={600} fill={theme.ink}
                    style={{ fontVariantNumeric: 'tabular-nums' }}
                  >
                    {d.value > 0 ? '+' : ''}{d.value.toFixed(1)}
                  </text>
                  <rect
                    x={0} y={y} width={W} height={rowH} fill="transparent"
                    onMouseMove={e => show(e, (
                      <>
                        <strong>{d.label}</strong><br />
                        Средний балл: {d.actual.toFixed(2)}<br />
                        Требуется: {d.target.toFixed(1)}<br />
                        {neg ? 'Дефицит' : 'Запас'}: {Math.abs(d.value).toFixed(2)}
                        {d.meta && <><br />{d.meta}</>}
                      </>
                    ))}
                    onMouseLeave={hide}
                  />
                </g>
              );
            })}
            {/* Neutral midpoint — the "meets requirement" line. */}
            <line x1={mid} x2={mid} y1={0} y2={data.length * rowH} stroke={theme.axis} strokeWidth={1} />
          </>
        )}
      </svg>
      <TooltipLayer tip={tip} />
    </div>
  );
}

/** Legend for the diverging chart — identity is never carried by color alone. */
export function DivergingLegend() {
  const theme = useChartTheme();
  const item = (color: string, text: string) => (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span style={{ width: 10, height: 10, borderRadius: 3, background: color, flexShrink: 0 }} />
      <span style={{ fontSize: 12, color: theme.inkSecondary, fontWeight: 400 }}>{text}</span>
    </span>
  );
  return (
    <span style={{ display: 'inline-flex', gap: 14 }}>
      {item(theme.negative, 'Ниже требования')}
      {item(theme.positive, 'Выше требования')}
    </span>
  );
}

/* ────────────────────────────────────────────────────────────────────────────
 * Trend area — one series, one axis
 * ──────────────────────────────────────────────────────────────────────────── */

export interface TrendDatum { label: string; value: number; note?: string }

export function TrendArea({ data, height = 220, unit = '' }: {
  data: TrendDatum[];
  height?: number;
  unit?: string;
}) {
  const theme = useChartTheme();
  const entered = useEntered();
  const [hostRef, W] = useMeasure();
  const { tip, show, hide } = useTooltip(hostRef);
  const gradId = useId().replace(/:/g, '');

  const padL = 38, padB = 26, padT = 14;
  const plotH = height - padB - padT;
  const plotW = Math.max(0, W - padL - 8);
  const max = niceMax(Math.max(...data.map(d => d.value), 1));
  const n = Math.max(data.length - 1, 1);

  const px = (i: number) => padL + (i / n) * plotW;
  const py = (v: number) => padT + plotH - (v / max) * plotH;

  const line = data.map((d, i) => `${i === 0 ? 'M' : 'L'}${px(i)},${py(d.value)}`).join(' ');
  const area = `${line} L${px(n)},${padT + plotH} L${px(0)},${padT + plotH} Z`;
  // Label only the ends and the middle — a tick under every month collides.
  const labelled = new Set([0, Math.floor(n / 2), n]);

  return (
    <div ref={hostRef} style={{ position: 'relative', width: '100%' }}>
      <svg width="100%" height={height} role="img" style={{ display: 'block' }}>
        {W > 0 && (
          <>
            <defs>
              <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={theme.series} stopOpacity={0.26} />
                <stop offset="100%" stopColor={theme.series} stopOpacity={0} />
              </linearGradient>
            </defs>
            {ticksFor(max).map(t => {
              const y = py(t);
              return (
                <g key={t}>
                  <line x1={padL} x2={W} y1={y} y2={y} stroke={theme.grid} strokeWidth={1} />
                  <text x={padL - 8} y={y} textAnchor="end" dominantBaseline="central" fontSize={11} fill={theme.inkMuted} style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {Math.round(t)}
                  </text>
                </g>
              );
            })}
            <path
              d={area} fill={`url(#${gradId})`}
              style={{ opacity: entered ? 1 : 0, transition: prefersReducedMotion() ? undefined : `opacity 700ms ${EASE} 180ms` }}
            />
            <path
              d={line} fill="none" stroke={theme.series} strokeWidth={2}
              strokeLinecap="round" strokeLinejoin="round" pathLength={1}
              style={{
                strokeDasharray: 1,
                strokeDashoffset: entered ? 0 : 1,
                transition: prefersReducedMotion() ? undefined : `stroke-dashoffset 900ms ${EASE}`,
              }}
            />
            {/* ≥8px endpoint marker with a 2px surface ring. */}
            <circle
              cx={px(n)} cy={py(data[data.length - 1]?.value ?? 0)} r={4.5}
              fill={theme.series} stroke={theme.surface} strokeWidth={2}
              style={{ opacity: entered ? 1 : 0, transition: prefersReducedMotion() ? undefined : 'opacity 400ms 780ms' }}
            />
            {data.map((d, i) => (
              <g key={d.label}>
                {labelled.has(i) && (
                  <text
                    x={px(i)} y={padT + plotH + 16}
                    textAnchor={i === 0 ? 'start' : i === n ? 'end' : 'middle'}
                    fontSize={11} fill={theme.inkMuted}
                  >
                    {d.label}
                  </text>
                )}
                <rect
                  x={px(i) - plotW / n / 2} y={padT} width={plotW / n} height={plotH} fill="transparent"
                  onMouseMove={e => show(e, <><strong>{d.label}</strong><br />{d.value}{unit && ` ${unit}`}{d.note && <><br />{d.note}</>}</>)}
                  onMouseLeave={hide}
                />
              </g>
            ))}
          </>
        )}
      </svg>
      <TooltipLayer tip={tip} />
    </div>
  );
}

/* ────────────────────────────────────────────────────────────────────────────
 * Meter — one ratio against its limit
 * ──────────────────────────────────────────────────────────────────────────── */

export function Meter({ label, value, max, caption, delay = 0 }: {
  label: ReactNode;
  value: number;
  max: number;
  caption?: ReactNode;
  delay?: number;
}) {
  const theme = useChartTheme();
  const entered = useEntered();
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0;

  return (
    <div style={{ marginBottom: 14 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, marginBottom: 6 }}>
        <span style={{ fontSize: 13, color: theme.ink, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {label}
        </span>
        <span style={{ fontSize: 12, color: theme.inkSecondary, whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>
          {caption}
        </span>
      </div>
      <div style={{ height: 8, borderRadius: R, background: theme.grid, overflow: 'hidden' }}>
        <div
          style={{
            height: '100%', width: entered ? `${pct}%` : 0, borderRadius: R,
            background: theme.series,
            transition: prefersReducedMotion() ? undefined : `width 700ms ${EASE} ${delay}ms`,
          }}
        />
      </div>
    </div>
  );
}
