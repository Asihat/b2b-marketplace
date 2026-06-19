<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Models\Company;
use App\Models\Order;
use App\Models\Product;
use App\Models\User;
use Illuminate\Http\JsonResponse;

class DashboardController extends Controller
{
    public function index(): JsonResponse
    {
        $paidStatuses = ['paid', 'processing', 'shipped', 'completed'];

        return response()->json([
            'stats' => [
                'users' => User::count(),
                'b2b_users' => User::where('type', 'b2b')->count(),
                'companies' => Company::count(),
                'products' => Product::count(),
                'active_products' => Product::where('is_active', true)->count(),
                'orders' => Order::count(),
                'pending_orders' => Order::where('status', 'pending')->count(),
                // Revenue is summed in each order's own currency; grouped below.
                'revenue_by_currency' => Order::whereIn('status', $paidStatuses)
                    ->selectRaw('currency_code, SUM(grand_total) as total')
                    ->groupBy('currency_code')
                    ->pluck('total', 'currency_code'),
            ],
            'recent_orders' => Order::with('user:id,name,email')
                ->latest()
                ->limit(8)
                ->get(['id', 'number', 'user_id', 'type', 'status', 'currency_code', 'grand_total', 'created_at']),
            'low_stock' => Product::where('is_active', true)
                ->orderBy('stock')
                ->limit(8)
                ->get(['id', 'sku', 'name', 'stock', 'min_order_qty']),
        ]);
    }
}
