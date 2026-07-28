import dayjs from 'dayjs';

// Formats an assessment period range. A one-day period (start === end) is shown
// as a single date instead of "10.09.2026 — 10.09.2026".
export function formatPeriodRange(start: string, end: string): string {
  const from = dayjs(start);
  const to = dayjs(end);
  if (from.isSame(to, 'day')) {
    return from.format('DD.MM.YYYY');
  }
  return `${from.format('DD.MM.YYYY')} — ${to.format('DD.MM.YYYY')}`;
}
