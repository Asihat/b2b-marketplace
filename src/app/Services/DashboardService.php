<?php

namespace App\Services;

use App\Enums\AccountType;
use App\Enums\OrderStatus;
use App\Enums\PaymentStatus;
use App\Models\Company;
use App\Models\Order;
use App\Models\Product;
use App\Models\User;
use Carbon\CarbonImmutable;
use Illuminate\Support\Facades\DB;

/**
 * Builds the admin dashboard read-model. Keeps reporting queries out of the
 * controller (single responsibility) and in one reusable place.
 *
 * Orders and payments are stored in the currency the buyer checked out with, so
 * every money figure aggregated here is grouped by currency in SQL and folded
 * into the base currency in PHP via CurrencyService::toBase().
 */
class DashboardService
{
    /** Day windows the dashboard can be asked for. */
    public const RANGES = [7, 30, 90];

    public const DEFAULT_RANGE = 30;

    /** Order statuses that count as realised revenue. */
    private const REVENUE_STATUSES = [
        OrderStatus::Paid,
        OrderStatus::Processing,
        OrderStatus::Shipped,
        OrderStatus::Completed,
    ];

    /** How many rows the "top" leaderboards return. */
    private const TOP_LIMIT = 5;

    public function __construct(private readonly CurrencyService $currency) {}

    public function overview(int $days = self::DEFAULT_RANGE): array
    {
        $days = in_array($days, self::RANGES, true) ? $days : self::DEFAULT_RANGE;

        $to = CarbonImmutable::now()->endOfDay();
        $from = $to->subDays($days - 1)->startOfDay();
        // The equally long window immediately before, for period-over-period deltas.
        $previousFrom = $from->subDays($days);
        $previousTo = $from->subSecond();

        $current = $this->windowMetrics($from, $to);
        $previous = $this->windowMetrics($previousFrom, $previousTo);
        $base = $this->currency->base();

        return [
            'range' => [
                'days' => $days,
                'from' => $from->toDateString(),
                'to' => $to->toDateString(),
            ],
            'base_currency' => ['code' => $base['code'], 'symbol' => $base['symbol']],
            'kpis' => [
                'revenue' => $this->kpi($current['revenue'], $previous['revenue']),
                'orders' => $this->kpi($current['orders'], $previous['orders']),
                'aov' => $this->kpi($current['aov'], $previous['aov']),
                'new_users' => $this->kpi($current['new_users'], $previous['new_users']),
            ],
            'stats' => $this->stats(),
            'timeseries' => $this->timeseries($from, $to),
            'orders_by_status' => $this->ordersByStatus($from, $to),
            'segments' => $this->segments($from, $to),
            'top_products' => $this->topProducts($from, $to),
            'top_categories' => $this->topCategories($from, $to),
            'gateways' => $this->gateways($from, $to),
            'stock' => $this->stock(),
            'recent_orders' => $this->recentOrders(),
            'low_stock' => $this->lowStock(),
        ];
    }

    /** All-time counters, independent of the selected range. */
    private function stats(): array
    {
        return [
            'users' => User::count(),
            'b2b_users' => User::where('type', AccountType::B2b->value)->count(),
            'companies' => Company::count(),
            'products' => Product::count(),
            'active_products' => Product::where('is_active', true)->count(),
            'orders' => Order::count(),
            'pending_orders' => Order::where('status', OrderStatus::Pending->value)->count(),
            'revenue_by_currency' => DB::table('orders')
                ->whereIn('status', $this->revenueStatuses())
                ->selectRaw('currency_code, SUM(grand_total) as total')
                ->groupBy('currency_code')
                ->pluck('total', 'currency_code'),
        ];
    }

    /**
     * Headline figures for one window. Revenue and AOV are in the base currency.
     *
     * @return array{revenue:float,orders:int,paid_orders:int,aov:float,new_users:int}
     */
    private function windowMetrics(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $rows = DB::table('orders')
            ->whereBetween('created_at', [$from, $to])
            ->whereIn('status', $this->revenueStatuses())
            ->selectRaw('currency_code, COUNT(*) as orders, SUM(grand_total) as total')
            ->groupBy('currency_code')
            ->get();

        $revenue = 0.0;
        $paidOrders = 0;

        foreach ($rows as $row) {
            $revenue += $this->currency->toBase((float) $row->total, $row->currency_code);
            $paidOrders += (int) $row->orders;
        }

        return [
            'revenue' => round($revenue, 2),
            'orders' => DB::table('orders')->whereBetween('created_at', [$from, $to])->count(),
            'paid_orders' => $paidOrders,
            'aov' => $paidOrders > 0 ? round($revenue / $paidOrders, 2) : 0.0,
            'new_users' => DB::table('users')->whereBetween('created_at', [$from, $to])->count(),
        ];
    }

    /** A metric plus its previous-window value and percentage change (null when there is no baseline). */
    private function kpi(float|int $current, float|int $previous): array
    {
        return [
            'current' => $current,
            'previous' => $previous,
            'change' => $previous > 0 ? round((($current - $previous) / $previous) * 100, 1) : null,
        ];
    }

    /**
     * Daily orders + base-currency revenue, with empty days filled in so the
     * chart always spans the whole window.
     */
    private function timeseries(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $day = $this->dayExpr('created_at');

        $rows = DB::table('orders')
            ->whereBetween('created_at', [$from, $to])
            ->selectRaw("{$day} as day, currency_code, COUNT(*) as orders, {$this->revenueSumExpr('grand_total')} as revenue")
            ->groupByRaw("{$day}, currency_code")
            ->get();

        $buckets = [];
        for ($date = $from; $date->lte($to); $date = $date->addDay()) {
            $buckets[$date->toDateString()] = ['date' => $date->toDateString(), 'orders' => 0, 'revenue' => 0.0];
        }

        foreach ($rows as $row) {
            if (! isset($buckets[$row->day])) {
                continue;
            }
            $buckets[$row->day]['orders'] += (int) $row->orders;
            $buckets[$row->day]['revenue'] += $this->currency->toBase((float) $row->revenue, $row->currency_code);
        }

        return array_values(array_map(
            fn (array $bucket) => [...$bucket, 'revenue' => round($bucket['revenue'], 2)],
            $buckets,
        ));
    }

    /** Order counts per status for the window; every status is present, even at zero. */
    private function ordersByStatus(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $counts = DB::table('orders')
            ->whereBetween('created_at', [$from, $to])
            ->selectRaw('status, COUNT(*) as count')
            ->groupBy('status')
            ->pluck('count', 'status');

        return array_map(
            fn (OrderStatus $status) => ['status' => $status->value, 'count' => (int) ($counts[$status->value] ?? 0)],
            OrderStatus::cases(),
        );
    }

    /** B2B vs B2C order volume and base-currency revenue for the window. */
    private function segments(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $rows = DB::table('orders')
            ->whereBetween('created_at', [$from, $to])
            ->selectRaw("type, currency_code, COUNT(*) as orders, {$this->revenueSumExpr('grand_total')} as revenue")
            ->groupByRaw('type, currency_code')
            ->get();

        $segments = [];
        foreach (AccountType::cases() as $type) {
            $segments[$type->value] = ['type' => $type->value, 'orders' => 0, 'revenue' => 0.0];
        }

        foreach ($rows as $row) {
            if (! isset($segments[$row->type])) {
                continue;
            }
            $segments[$row->type]['orders'] += (int) $row->orders;
            $segments[$row->type]['revenue'] += $this->currency->toBase((float) $row->revenue, $row->currency_code);
        }

        return array_values(array_map(
            fn (array $segment) => [...$segment, 'revenue' => round($segment['revenue'], 2)],
            $segments,
        ));
    }

    /** Best sellers by base-currency revenue. Uses the order-item snapshot, so deleted products still appear. */
    private function topProducts(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $rows = $this->soldItems($from, $to)
            ->selectRaw('order_items.product_id, order_items.sku, order_items.name, orders.currency_code,
                         SUM(order_items.quantity) as qty, SUM(order_items.line_total) as revenue')
            ->groupByRaw('order_items.product_id, order_items.sku, order_items.name, orders.currency_code')
            ->get();

        $products = [];
        foreach ($rows as $row) {
            // Key on the SKU: it survives a deleted product (product_id goes null).
            $key = $row->sku;
            $products[$key] ??= ['id' => $row->product_id, 'sku' => $row->sku, 'name' => $row->name, 'qty' => 0, 'revenue' => 0.0];
            $products[$key]['qty'] += (int) $row->qty;
            $products[$key]['revenue'] += $this->currency->toBase((float) $row->revenue, $row->currency_code);
        }

        return $this->topByRevenue($products);
    }

    /** Revenue per category for the window. Items whose product or category is gone fall into "Uncategorized". */
    private function topCategories(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $rows = $this->soldItems($from, $to)
            ->leftJoin('products', 'products.id', '=', 'order_items.product_id')
            ->leftJoin('categories', 'categories.id', '=', 'products.category_id')
            ->selectRaw('categories.id as category_id, categories.name as category_name, orders.currency_code,
                         SUM(order_items.quantity) as qty, SUM(order_items.line_total) as revenue')
            ->groupByRaw('categories.id, categories.name, orders.currency_code')
            ->get();

        $categories = [];
        foreach ($rows as $row) {
            $key = $row->category_id ?? 0;
            $categories[$key] ??= [
                'id' => $row->category_id,
                'name' => $row->category_name ?? 'Uncategorized',
                'qty' => 0,
                'revenue' => 0.0,
            ];
            $categories[$key]['qty'] += (int) $row->qty;
            $categories[$key]['revenue'] += $this->currency->toBase((float) $row->revenue, $row->currency_code);
        }

        return $this->topByRevenue($categories);
    }

    /** Completed payments per gateway for the window, in the base currency. */
    private function gateways(CarbonImmutable $from, CarbonImmutable $to): array
    {
        $rows = DB::table('payments')
            ->where('status', PaymentStatus::Completed->value)
            ->whereBetween('created_at', [$from, $to])
            ->selectRaw('gateway, currency_code, COUNT(*) as count, SUM(amount) as total')
            ->groupByRaw('gateway, currency_code')
            ->get();

        $gateways = [];
        foreach ($rows as $row) {
            $gateways[$row->gateway] ??= ['gateway' => $row->gateway, 'count' => 0, 'amount' => 0.0];
            $gateways[$row->gateway]['count'] += (int) $row->count;
            $gateways[$row->gateway]['amount'] += $this->currency->toBase((float) $row->total, $row->currency_code);
        }

        usort($gateways, fn (array $a, array $b) => $b['amount'] <=> $a['amount']);

        return array_map(fn (array $g) => [...$g, 'amount' => round($g['amount'], 2)], array_values($gateways));
    }

    /** Catalog health counters that need attention rather than a trend. */
    private function stock(): array
    {
        return [
            'out_of_stock' => Product::where('is_active', true)->where('stock', 0)->count(),
            'low_stock' => Product::where('is_active', true)
                ->where('stock', '>', 0)
                ->whereColumn('stock', '<=', 'min_order_qty')
                ->count(),
            'inactive' => Product::where('is_active', false)->count(),
        ];
    }

    private function recentOrders()
    {
        return Order::with('user:id,name,email')
            ->latest()
            ->limit(8)
            ->get(['id', 'number', 'user_id', 'type', 'status', 'currency_code', 'grand_total', 'created_at']);
    }

    private function lowStock()
    {
        return Product::where('is_active', true)
            ->orderBy('stock')
            ->limit(8)
            ->get(['id', 'sku', 'name', 'stock', 'min_order_qty']);
    }

    /** Order items belonging to revenue-bearing orders placed in the window. */
    private function soldItems(CarbonImmutable $from, CarbonImmutable $to): \Illuminate\Database\Query\Builder
    {
        return DB::table('order_items')
            ->join('orders', 'orders.id', '=', 'order_items.order_id')
            ->whereBetween('orders.created_at', [$from, $to])
            ->whereIn('orders.status', $this->revenueStatuses());
    }

    /** Round, sort by revenue and cap a keyed leaderboard. */
    private function topByRevenue(array $rows): array
    {
        usort($rows, fn (array $a, array $b) => $b['revenue'] <=> $a['revenue']);

        return array_map(
            fn (array $row) => [...$row, 'revenue' => round($row['revenue'], 2)],
            array_slice(array_values($rows), 0, self::TOP_LIMIT),
        );
    }

    /** @return list<string> */
    private function revenueStatuses(): array
    {
        return array_column(self::REVENUE_STATUSES, 'value');
    }

    /**
     * SUM() that only counts revenue-bearing rows. The statuses are enum
     * constants, never user input, so they are safe to inline.
     */
    private function revenueSumExpr(string $column): string
    {
        $statuses = "'".implode("','", $this->revenueStatuses())."'";

        return "SUM(CASE WHEN status IN ({$statuses}) THEN {$column} ELSE 0 END)";
    }

    /** Portable "truncate a timestamp to a YYYY-MM-DD string" expression (Postgres in prod, SQLite in tests). */
    private function dayExpr(string $column): string
    {
        return match (DB::connection()->getDriverName()) {
            'pgsql' => "TO_CHAR({$column}, 'YYYY-MM-DD')",
            'sqlite' => "strftime('%Y-%m-%d', {$column})",
            'sqlsrv' => "FORMAT({$column}, 'yyyy-MM-dd')",
            default => "DATE_FORMAT({$column}, '%Y-%m-%d')",
        };
    }
}
