<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\UpdateOrderStatusRequest;
use App\Models\Order;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class OrderController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $orders = Order::query()
            ->with(['user:id,name,email', 'company:id,name'])
            ->when($request->query('status'), fn ($q, $s) => $q->where('status', $s))
            ->when($request->query('type'), fn ($q, $t) => $q->where('type', $t))
            ->numberLike($request->query('search'))
            ->latest()
            ->paginate(20);

        return response()->json($orders);
    }

    public function show(Order $order): JsonResponse
    {
        return response()->json($order->load(['items', 'payments', 'user:id,name,email', 'company:id,name']));
    }

    public function updateStatus(UpdateOrderStatusRequest $request, Order $order): JsonResponse
    {
        $order->update($request->validated());

        return response()->json($order->fresh()->load('items'));
    }
}
