// Rounds a fractional mark to an integer 1–10 for interpretation-template
// lookup. Half rounds DOWN per product rule: 5.5 → 5, 5.6 → 6.
export function roundMark(x: number): number {
  const floor = Math.floor(x);
  return x - floor > 0.5 ? floor + 1 : floor;
}
