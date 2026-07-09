import { useEffect, useRef, useState, type ReactNode } from "react";
import { adminApi, type Dashboard as DashboardData, type Kpi } from "../../api";
import { PageHeader, StatusBadge } from "../../components/ui";
import { BarList, StackedShareBar, TrendChart } from "../../components/admin/charts";
import { formatCompact, formatCompactMoney, formatNumber, formatShortDate } from "../../lib/format";

const RANGES = [7, 30, 90];

type Metric = "revenue" | "orders";

/** One headline number: value, period-over-period delta, optional caption. */
function StatTile({ label, value, kpi, caption }: { label: string; value: string; kpi?: Kpi; caption?: string }) {
  return (
    <div className="card card-hover p-4">
      <div className="text-xs text-slate-400">{label}</div>
      {/* Proportional figures: tabular-nums makes large standalone numbers look loose. */}
      <div className="mt-1.5 text-2xl font-bold text-slate-900">{value}</div>
      {kpi ? <Delta change={kpi.change} caption={caption} /> : caption && <Caption>{caption}</Caption>}
    </div>
  );
}

function Delta({ change, caption }: { change: number | null; caption?: string }) {
  if (change === null) return <Caption>{caption ?? "no prior data"}</Caption>;

  const up = change >= 0;
  return (
    <div className="mt-1.5 flex items-center gap-1.5 text-xs">
      <span className={`font-medium ${up ? "text-emerald-700" : "text-rose-600"}`}>
        {up ? "↑" : "↓"} {Math.abs(change)}%
      </span>
      {caption && <span className="text-slate-400">{caption}</span>}
    </div>
  );
}

function Caption({ children }: { children: ReactNode }) {
  return <div className="mt-1.5 text-xs text-slate-400">{children}</div>;
}

function Card({ title, action, children }: { title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <div className="card p-4">
      <div className="mb-4 flex items-center justify-between gap-2">
        <h2 className="font-semibold text-slate-700">{title}</h2>
        {action}
      </div>
      {children}
    </div>
  );
}

/** Segmented pill control, matching the filter pills on the Products page. */
function Segmented<T extends string | number>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
}) {
  return (
    <div className="inline-flex rounded-lg bg-slate-100 p-0.5">
      {options.map((option) => (
        <button
          key={option.value}
          onClick={() => onChange(option.value)}
          aria-pressed={option.value === value}
          className={`cursor-pointer rounded-[6px] px-2.5 py-1 text-xs font-medium transition ${
            option.value === value ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700"
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

export function Dashboard() {
  const [days, setDays] = useState(30);
  const [metric, setMetric] = useState<Metric>("revenue");
  const [asTable, setAsTable] = useState(false);
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const requestId = useRef(0);

  useEffect(() => {
    const id = ++requestId.current;
    setLoading(true);

    adminApi
      .dashboard(days)
      .then((result) => {
        if (id !== requestId.current) return; // a newer range won the race
        setData(result);
        setError("");
      })
      .catch((e: Error) => id === requestId.current && setError(e.message))
      .finally(() => id === requestId.current && setLoading(false));
  }, [days]);

  if (error) return <p className="text-rose-600">{error}</p>;
  if (!data) return <p className="text-slate-400">Loading dashboard…</p>;

  const { kpis, stats, stock, base_currency: base } = data;
  const money = (value: number) => formatCompactMoney(value, base.symbol);
  const exactMoney = (value: number) =>
    `${base.symbol}${value.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;

  const since = `vs previous ${days} days`;
  const revenueSeries = data.timeseries.map((point) => ({ date: point.date, value: point[metric] }));
  const quiet = kpis.orders.current === 0;

  return (
    <div>
      <PageHeader
        title="Dashboard"
        action={
          <Segmented
            value={days}
            onChange={setDays}
            options={RANGES.map((r) => ({ value: r, label: `${r}d` }))}
          />
        }
      />

      <p className="-mt-4 mb-6 text-xs text-slate-400">
        {formatShortDate(data.range.from)} – {formatShortDate(data.range.to)} · amounts in {base.code}
      </p>

      {/* Refetching holds the previous render at reduced opacity — no skeleton flash. */}
      <div className={loading ? "opacity-60 transition-opacity" : "transition-opacity"}>
        <div className="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatTile label="Revenue" value={money(kpis.revenue.current)} kpi={kpis.revenue} caption={since} />
          <StatTile label="Orders" value={formatNumber(kpis.orders.current)} kpi={kpis.orders} caption={since} />
          <StatTile label="Avg. order value" value={money(kpis.aov.current)} kpi={kpis.aov} caption={since} />
          <StatTile label="New users" value={formatNumber(kpis.new_users.current)} kpi={kpis.new_users} caption={since} />
        </div>

        <div className="mb-4 grid gap-4 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <Card
              title={metric === "revenue" ? `Revenue per day (${base.code})` : "Orders per day"}
              action={
                <div className="flex items-center gap-2">
                  <Segmented
                    value={metric}
                    onChange={setMetric}
                    options={[
                      { value: "revenue", label: "Revenue" },
                      { value: "orders", label: "Orders" },
                    ]}
                  />
                  <button
                    onClick={() => setAsTable((v) => !v)}
                    className="cursor-pointer rounded-lg px-2 py-1 text-xs font-medium text-slate-500 transition hover:bg-slate-100 hover:text-slate-700"
                  >
                    {asTable ? "Chart" : "Table"}
                  </button>
                </div>
              }
            >
              {asTable ? (
                <TrendTable data={data} exactMoney={exactMoney} />
              ) : (
                <TrendChart
                  points={revenueSeries}
                  seriesLabel={metric === "revenue" ? "Revenue" : "Orders"}
                  formatValue={(v) => (metric === "revenue" ? exactMoney(v) : `${formatNumber(v)} orders`)}
                  formatTick={(v) => (metric === "revenue" ? money(v) : formatCompact(v))}
                />
              )}
              {quiet && <p className="mt-3 text-xs text-slate-400">No orders were placed in this range.</p>}
            </Card>
          </div>

          <Card title="B2B vs B2C revenue">
            <StackedShareBar
              formatValue={money}
              segments={data.segments.map((segment) => ({
                key: segment.type,
                label: segment.type.toUpperCase(),
                value: segment.revenue,
                caption: `${formatNumber(segment.orders)} orders`,
              }))}
            />
          </Card>
        </div>

        <div className="mb-4 grid gap-4 lg:grid-cols-3">
          <Card title="Orders by status">
            <BarList
              formatValue={formatNumber}
              items={data.orders_by_status.map((row) => ({
                key: row.status,
                label: <StatusBadge status={row.status} />,
                value: row.count,
              }))}
            />
          </Card>

          <Card title="Payments by gateway">
            {data.gateways.length ? (
              <BarList
                formatValue={money}
                items={data.gateways.map((row) => ({
                  key: row.gateway,
                  label: <span className="capitalize text-slate-600">{row.gateway}</span>,
                  value: row.amount,
                  hint: `${formatNumber(row.count)}×`,
                }))}
              />
            ) : (
              <p className="text-sm text-slate-400">No payments captured in this range.</p>
            )}
          </Card>

          <Card title="Catalog health">
            <ul className="space-y-3 text-sm">
              <HealthRow label="Out of stock" value={stock.out_of_stock} color="#d03b3b" />
              <HealthRow label="At or below MOQ" value={stock.low_stock} color="#fab219" />
              <HealthRow label="Hidden products" value={stock.inactive} color="#94a3b8" />
              <HealthRow label="Pending orders" value={stats.pending_orders} color="#94a3b8" />
            </ul>
          </Card>
        </div>

        <div className="mb-4 grid gap-4 lg:grid-cols-2">
          <Card title="Top products">
            <Leaderboard
              rows={data.top_products.map((p) => ({
                key: p.sku,
                name: p.name,
                meta: p.sku,
                qty: p.qty,
                revenue: p.revenue,
              }))}
              exactMoney={exactMoney}
            />
          </Card>

          <Card title="Top categories">
            <Leaderboard
              rows={data.top_categories.map((c) => ({
                key: String(c.id ?? c.name),
                name: c.name,
                qty: c.qty,
                revenue: c.revenue,
              }))}
              exactMoney={exactMoney}
            />
          </Card>
        </div>

        <div className="mb-4 grid gap-4 lg:grid-cols-2">
          <Card title="Recent orders">
            <table className="w-full text-sm">
              <tbody>
                {data.recent_orders.map((order) => (
                  <tr key={order.id} className="border-t border-slate-100">
                    <td className="py-2 font-medium">{order.number}</td>
                    <td className="text-slate-500">{order.user?.name ?? "—"}</td>
                    <td className="text-right tabular-nums">
                      {Number(order.grand_total).toFixed(2)} {order.currency_code}
                    </td>
                    <td className="pl-2 text-right">
                      <StatusBadge status={order.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>

          <Card title="Lowest stock">
            <table className="w-full text-sm">
              <tbody>
                {data.low_stock.map((product) => (
                  <tr key={product.id} className="border-t border-slate-100">
                    <td className="py-2 font-medium">{product.name}</td>
                    <td className="text-slate-400">{product.sku}</td>
                    <td
                      className={`text-right tabular-nums ${
                        product.stock <= product.min_order_qty ? "font-semibold text-red-600" : ""
                      }`}
                    >
                      {product.stock}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </div>

        <Card title="All time">
          <div className="grid grid-cols-2 gap-4 text-sm lg:grid-cols-5">
            <Total label="Users" value={`${formatNumber(stats.b2b_users)} B2B / ${formatNumber(stats.users)}`} />
            <Total label="Companies" value={formatNumber(stats.companies)} />
            <Total label="Products (active)" value={`${formatNumber(stats.active_products)}/${formatNumber(stats.products)}`} />
            <Total label="Orders" value={formatNumber(stats.orders)} />
            <Total
              label="Revenue booked"
              value={
                Object.entries(stats.revenue_by_currency)
                  .map(([code, total]) => `${Number(total).toFixed(0)} ${code}`)
                  .join(" · ") || "—"
              }
            />
          </div>
        </Card>
      </div>
    </div>
  );
}

function Total({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-slate-400">{label}</div>
      <div className="mt-1 font-semibold text-slate-900">{value}</div>
    </div>
  );
}

/** Status colours never carry meaning alone — each ships with its label. */
function HealthRow({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <li className="flex items-center gap-2">
      <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: value ? color : "#e2e8f0" }} />
      <span className="text-slate-600">{label}</span>
      <span className="ml-auto font-semibold text-slate-900 tabular-nums">{formatNumber(value)}</span>
    </li>
  );
}

function Leaderboard({
  rows,
  exactMoney,
}: {
  rows: { key: string; name: string; meta?: string; qty: number; revenue: number }[];
  exactMoney: (value: number) => string;
}) {
  if (!rows.length) return <p className="text-sm text-slate-400">Nothing sold in this range.</p>;

  return (
    <table className="w-full text-sm">
      <tbody>
        {rows.map((row) => (
          <tr key={row.key} className="border-t border-slate-100 first:border-0">
            <td className="py-2">
              <div className="font-medium text-slate-800">{row.name}</div>
              {row.meta && <div className="text-xs text-slate-400">{row.meta}</div>}
            </td>
            <td className="text-right text-xs text-slate-400 tabular-nums">{formatNumber(row.qty)} sold</td>
            <td className="pl-3 text-right font-medium tabular-nums">{exactMoney(row.revenue)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/** The chart's table twin — every plotted value stays reachable without hover. */
function TrendTable({ data, exactMoney }: { data: DashboardData; exactMoney: (value: number) => string }) {
  return (
    <div className="max-h-[240px] overflow-y-auto">
      <table className="w-full text-sm">
        <thead className="sticky top-0 bg-white text-left text-xs text-slate-400">
          <tr>
            <th className="py-1 font-medium">Date</th>
            <th className="py-1 text-right font-medium">Orders</th>
            <th className="py-1 text-right font-medium">Revenue</th>
          </tr>
        </thead>
        <tbody>
          {data.timeseries.map((point) => (
            <tr key={point.date} className="border-t border-slate-100">
              <td className="py-1.5 text-slate-600">{formatShortDate(point.date)}</td>
              <td className="py-1.5 text-right tabular-nums">{formatNumber(point.orders)}</td>
              <td className="py-1.5 text-right tabular-nums">{exactMoney(point.revenue)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
