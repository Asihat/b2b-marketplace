<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Http\Requests\StoreOrderRequest;
use App\Models\Order;
use App\Services\OrderService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use RuntimeException;

class OrderController extends Controller
{
    public function __construct(private readonly OrderService $orders) {}

    public function index(Request $request): JsonResponse
    {
        $orders = Order::with(['items', 'payment'])
            ->visibleTo($request->user())
            ->latest()
            ->paginate(20);

        return response()->json($orders);
    }

    public function store(StoreOrderRequest $request): JsonResponse
    {
        try {
            $order = $this->orders->place($request->user(), $request->validated()['items'], $request->validated());
        } catch (RuntimeException $e) {
            return response()->json(['message' => $e->getMessage()], 422);
        }

        return response()->json($order->load('items'), 201);
    }

    public function show(Request $request, Order $order): JsonResponse
    {
        $this->authorize('view', $order);

        return response()->json($order->load(['items', 'payments']));
    }
}
