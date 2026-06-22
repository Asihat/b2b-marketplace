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
        $user = $request->user();

        $query = Order::with(['items', 'payment'])->latest();

        // B2B buyers see their whole company's orders; everyone else, their own.
        $user->isB2b() && $user->company_id
            ? $query->where('company_id', $user->company_id)
            : $query->where('user_id', $user->id);

        return response()->json($query->paginate(20));
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
        $this->authorizeOrder($request, $order);

        return response()->json($order->load(['items', 'payments']));
    }

    private function authorizeOrder(Request $request, Order $order): void
    {
        $user = $request->user();
        $ownsOrder = $order->user_id === $user->id
            || ($user->isB2b() && $order->company_id && $order->company_id === $user->company_id);

        abort_unless($ownsOrder || $user->isAdmin(), 403, 'Not your order.');
    }
}
