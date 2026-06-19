<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Models\Order;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class OrderController extends Controller
{
    public const STATUSES = ['pending', 'paid', 'processing', 'shipped', 'completed', 'cancelled'];

    public function index(Request $request): JsonResponse
    {
        $orders = Order::query()
            ->with(['user:id,name,email', 'company:id,name'])
            ->when($request->query('status'), fn ($q, $s) => $q->where('status', $s))
            ->when($request->query('type'), fn ($q, $t) => $q->where('type', $t))
            ->when($request->query('search'), fn ($q, $s) => $q->where('number', 'ilike', "%{$s}%"))
            ->latest()
            ->paginate(20);

        return response()->json($orders);
    }

    public function show(Order $order): JsonResponse
    {
        return response()->json($order->load(['items', 'payments', 'user:id,name,email', 'company:id,name']));
    }

    public function updateStatus(Request $request, Order $order): JsonResponse
    {
        $data = $request->validate([
            'status' => ['required', 'in:'.implode(',', self::STATUSES)],
        ]);

        $order->update($data);

        return response()->json($order->fresh()->load('items'));
    }
}
