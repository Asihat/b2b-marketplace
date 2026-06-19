<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Order;
use App\Services\OrderService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use RuntimeException;

class OrderController extends Controller
{
    public function __construct(protected OrderService $orders) {}

    public function index(Request $request): JsonResponse
    {
        $query = Order::with(['items', 'payment'])->latest();

        // B2B managers see the whole company's orders; everyone else sees their own.
        $user = $request->user();
        if ($user->isB2b() && $user->company_id) {
            $query->where('company_id', $user->company_id);
        } else {
            $query->where('user_id', $user->id);
        }

        return response()->json($query->paginate(20));
    }

    public function store(Request $request): JsonResponse
    {
        $data = $request->validate([
            'items' => ['required', 'array', 'min:1'],
            'items.*.product_id' => ['required', 'integer', 'exists:products,id'],
            'items.*.quantity' => ['required', 'integer', 'min:1'],
            'currency_code' => ['nullable', 'string', 'size:3'],
            'shipping_address' => ['nullable', 'string'],
            'notes' => ['nullable', 'string'],
            'tax_rate' => ['nullable', 'numeric', 'min:0', 'max:1'],
        ]);

        try {
            $order = $this->orders->place($request->user(), $data['items'], $data);
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

    protected function authorizeOrder(Request $request, Order $order): void
    {
        $user = $request->user();
        $owns = $order->user_id === $user->id
            || ($user->isB2b() && $order->company_id && $order->company_id === $user->company_id);

        abort_unless($owns || $user->isAdmin(), 403, 'Not your order.');
    }
}
