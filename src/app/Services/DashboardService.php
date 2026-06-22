<?php

namespace App\Services;

use App\Enums\AccountType;
use App\Enums\OrderStatus;
use App\Models\Company;
use App\Models\Order;
use App\Models\Product;
use App\Models\User;

/**
 * Builds the admin dashboard read-model. Keeps reporting queries out of the
 * controller (single responsibility) and in one reusable place.
 */
class DashboardService
{
    /** Order statuses that count as realised revenue. */
    private const REVENUE_STATUSES = [
        OrderStatus::Paid,
        OrderStatus::Processing,
        OrderStatus::Shipped,
        OrderStatus::Completed,
    ];

    public function overview(): array
    {
        return [
            'stats' => $this->stats(),
            'recent_orders' => $this->recentOrders(),
            'low_stock' => $this->lowStock(),
        ];
    }

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
            'revenue_by_currency' => Order::whereIn('status', array_column(self::REVENUE_STATUSES, 'value'))
                ->selectRaw('currency_code, SUM(grand_total) as total')
                ->groupBy('currency_code')
                ->pluck('total', 'currency_code'),
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
}
