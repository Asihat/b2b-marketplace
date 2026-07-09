import { useState, type ReactNode } from "react";
import { useElementWidth } from "../../hooks/useElementWidth";

/**
 * Chart tokens. The accent is the brand hue; the two categorical slots were
 * validated for colour-vision deficiency and ≥3:1 contrast on the white card
 * surface (worst adjacent ΔE 93.7).
 */
export const VIZ = {
  accent: "#4f46e5",
  surface: "#ffffff",
  grid: "#e2e8f0",
  axis: "#94a3b8",
  series: ["#4f46e5", "#199e70"] as const,
};

/** Rounds an axis maximum up to a clean tick value (1, 2, 2.5, 5 × 10ⁿ). */
function niceMax(max: number): number {
  if (max <= 0) return 1;

  const magnitude = 10 ** Math.floor(Math.log10(max));
  const normalized = max / magnitude;
  const step = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 2.5 ? 2.5 : normalized <= 5 ? 5 : 10;

  return step * magnitude;
}

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max);

export interface TrendPoint {
  date: string;
  value: number;
}

const PAD = { top: 16, right: 18, bottom: 26, left: 56 };
const HEIGHT = 240;
const TICKS = 4;

/**
 * Single-series area + line chart with a hover/keyboard crosshair. One series,
 * so no legend: the card title names what is plotted.
 */
export function TrendChart({
  points,
  formatValue,
  formatTick,
  seriesLabel,
}: {
  points: TrendPoint[];
  formatValue: (value: number) => string;
  formatTick: (value: number) => string;
  seriesLabel: string;
}) {
  const [wrapperRef, width] = useElementWidth<HTMLDivElement>();
  const [active, setActive] = useState<number | null>(null);

  const innerWidth = Math.max(width - PAD.left - PAD.right, 0);
  const innerHeight = HEIGHT - PAD.top - PAD.bottom;
  const max = niceMax(Math.max(...points.map((p) => p.value), 0));
  const baseline = PAD.top + innerHeight;
  const last = points.length - 1;

  const xOf = (index: number) =>
    points.length <= 1 ? PAD.left + innerWidth / 2 : PAD.left + (index / last) * innerWidth;
  const yOf = (value: number) => baseline - (value / max) * innerHeight;

  const line = points.map((p, i) => `${i === 0 ? "M" : "L"}${xOf(i)},${yOf(p.value)}`).join(" ");
  const area = points.length ? `${line} L${xOf(last)},${baseline} L${xOf(0)},${baseline} Z` : "";

  // Five evenly spaced date labels, endpoints included.
  const labelIndexes = points.length
    ? [...new Set([0, 1, 2, 3, 4].map((i) => Math.round((i / 4) * last)))]
    : [];

  function pointAt(clientX: number, bounds: DOMRect) {
    if (points.length < 2 || innerWidth <= 0) return 0;
    const ratio = (clientX - bounds.left - PAD.left) / innerWidth;
    return clamp(Math.round(ratio * last), 0, last);
  }

  const activePoint = active !== null ? points[active] : null;

  return (
    <div ref={wrapperRef} className="relative">
      {width > 0 && (
        <svg
          width={width}
          height={HEIGHT}
          role="img"
          aria-label={`${seriesLabel} per day`}
          tabIndex={0}
          className="touch-none focus-visible:rounded-lg"
          onMouseMove={(e) => setActive(pointAt(e.clientX, e.currentTarget.getBoundingClientRect()))}
          onMouseLeave={() => setActive(null)}
          onKeyDown={(e) => {
            if (e.key === "ArrowRight") setActive((i) => clamp((i ?? -1) + 1, 0, last));
            else if (e.key === "ArrowLeft") setActive((i) => clamp((i ?? last + 1) - 1, 0, last));
            else if (e.key === "Escape") setActive(null);
            else return;
            e.preventDefault();
          }}
        >
          {/* Hairline gridlines + y ticks */}
          {Array.from({ length: TICKS + 1 }, (_, i) => {
            const value = (max / TICKS) * i;
            const y = yOf(value);
            return (
              <g key={i}>
                <line x1={PAD.left} x2={width - PAD.right} y1={y} y2={y} stroke={VIZ.grid} strokeWidth={1} />
                <text
                  x={PAD.left - 10}
                  y={y + 4}
                  textAnchor="end"
                  fontSize={11}
                  fill={VIZ.axis}
                  style={{ fontVariantNumeric: "tabular-nums" }}
                >
                  {formatTick(value)}
                </text>
              </g>
            );
          })}

          {labelIndexes.map((i) => (
            <text key={i} x={xOf(i)} y={HEIGHT - 8} textAnchor="middle" fontSize={11} fill={VIZ.axis}>
              {formatShort(points[i].date)}
            </text>
          ))}

          <path d={area} fill={VIZ.accent} fillOpacity={0.1} />
          <path d={line} fill="none" stroke={VIZ.accent} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />

          {/* Endpoint marker: the one direct label the chart carries. */}
          {points.length > 0 && (
            <circle
              cx={xOf(last)}
              cy={yOf(points[last].value)}
              r={4}
              fill={VIZ.accent}
              stroke={VIZ.surface}
              strokeWidth={2}
            />
          )}

          {activePoint && (
            <g>
              <line
                x1={xOf(active!)}
                x2={xOf(active!)}
                y1={PAD.top}
                y2={baseline}
                stroke={VIZ.axis}
                strokeWidth={1}
              />
              <circle
                cx={xOf(active!)}
                cy={yOf(activePoint.value)}
                r={5}
                fill={VIZ.accent}
                stroke={VIZ.surface}
                strokeWidth={2}
              />
            </g>
          )}
        </svg>
      )}

      {activePoint && (
        <div
          className="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full rounded-lg bg-slate-900 px-2.5 py-1.5 text-xs text-white shadow-lg"
          style={{
            left: clamp(xOf(active!), 60, Math.max(width - 60, 60)),
            top: yOf(activePoint.value) - 10,
          }}
        >
          <div className="font-semibold">{formatValue(activePoint.value)}</div>
          <div className="text-slate-300">{formatShort(activePoint.date)}</div>
        </div>
      )}
    </div>
  );
}

function formatShort(isoDate: string): string {
  return new Date(`${isoDate}T00:00:00`).toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export interface BarItem {
  key: string;
  label: ReactNode;
  value: number;
  hint?: string;
}

/**
 * Horizontal bars sharing one scale. Nominal categories, so every bar wears the
 * same hue — length alone encodes magnitude — and the value rides the tip.
 */
export function BarList({ items, formatValue }: { items: BarItem[]; formatValue: (value: number) => string }) {
  const max = Math.max(...items.map((i) => i.value), 1);

  return (
    <ul className="space-y-2.5">
      {items.map((item) => (
        <li key={item.key} className="flex items-center gap-3">
          <div className="w-24 shrink-0 text-xs text-slate-500">{item.label}</div>
          <div className="flex flex-1 items-center gap-2">
            <div
              className="h-5 rounded-r-[4px]"
              style={{ width: `${(item.value / max) * 100}%`, backgroundColor: VIZ.accent }}
            />
            <span className="text-xs font-medium text-slate-700 tabular-nums">{formatValue(item.value)}</span>
            {item.hint && <span className="text-xs text-slate-400">{item.hint}</span>}
          </div>
        </li>
      ))}
    </ul>
  );
}

export interface ShareSegment {
  key: string;
  label: string;
  value: number;
  caption: string;
}

/**
 * Part-to-whole bar. Segments are separated by a 2px surface gap (never a
 * stroke) and identity comes from the legend below, never from colour alone.
 */
export function StackedShareBar({
  segments,
  formatValue,
}: {
  segments: ShareSegment[];
  formatValue: (value: number) => string;
}) {
  const total = segments.reduce((sum, s) => sum + s.value, 0);
  if (total <= 0) return <p className="text-sm text-slate-400">No revenue in this range.</p>;

  const shares = segments.map((segment, i) => ({
    ...segment,
    color: VIZ.series[i % VIZ.series.length],
    share: (segment.value / total) * 100,
  }));

  return (
    <div>
      <div className="flex h-6 gap-[2px] overflow-hidden rounded-[4px]">
        {shares.map((s) => (
          <div
            key={s.key}
            className="grid h-full place-items-center"
            style={{ width: `${s.share}%`, backgroundColor: s.color }}
          >
            {/* Only label inside the fill when the text comfortably fits. */}
            {s.share >= 18 && (
              <span className="px-2 text-[11px] font-semibold text-white">{s.share.toFixed(0)}%</span>
            )}
          </div>
        ))}
      </div>

      <ul className="mt-4 space-y-2">
        {shares.map((s) => (
          <li key={s.key} className="flex items-center gap-2 text-sm">
            <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: s.color }} />
            <span className="font-medium text-slate-700">{s.label}</span>
            <span className="text-xs text-slate-400">{s.caption}</span>
            <span className="ml-auto font-medium text-slate-900 tabular-nums">{formatValue(s.value)}</span>
            <span className="w-10 text-right text-xs text-slate-400 tabular-nums">{s.share.toFixed(0)}%</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
