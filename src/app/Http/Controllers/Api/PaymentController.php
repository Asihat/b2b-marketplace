<?php

namespace App\Http\Controllers\Api;

use App\Enums\OrderStatus;
use App\Enums\PaymentStatus;
use App\Http\Controllers\Controller;
use App\Models\Order;
use App\Models\Payment;
use App\Payments\PaymentManager;
use App\Services\OrderService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PaymentController extends Controller
{
    public function __construct(
        private readonly OrderService $orders,
        private readonly PaymentManager $gateways,
    ) {}

    /** List available payment gateways. */
    public function gateways(): JsonResponse
    {
        return response()->json([
            'default' => config('payments.default'),
            'available' => $this->gateways->available(),
        ]);
    }

    /** Initiate payment for an order. */
    public function pay(Request $request, Order $order): JsonResponse
    {
        $this->authorizeOrder($request, $order);

        if ($order->status->isSettled()) {
            return response()->json(['message' => 'Order is already paid.'], 422);
        }

        $data = $request->validate([
            'gateway' => ['nullable', 'string', 'in:'.implode(',', $this->gateways->available())],
        ]);

        $payment = $this->orders->pay($order, $data['gateway'] ?? null);

        return response()->json([
            'order' => $order->fresh(),
            'payment' => $payment,
        ], 201);
    }

    /**
     * Provider webhook/callback endpoint (public). Confirms async payments
     * such as bank transfers. Signature verification belongs in each driver.
     */
    public function callback(Request $request, string $gateway): JsonResponse
    {
        $result = $this->gateways->driver($gateway)->callback($request->all());
        $completed = $result->status === PaymentStatus::Completed->value;

        $payment = $result->reference
            ? Payment::where('reference', $result->reference)->first()
            : null;

        if ($payment) {
            $payment->update([
                'status' => $result->status,
                'paid_at' => $completed ? now() : $payment->paid_at,
                'payload' => $result->payload,
            ]);

            if ($completed) {
                $payment->order->update(['status' => OrderStatus::Paid]);
            }
        }

        return response()->json(['status' => $result->status]);
    }

    private function authorizeOrder(Request $request, Order $order): void
    {
        $user = $request->user();
        $ownsOrder = $order->user_id === $user->id
            || ($user->isB2b() && $order->company_id === $user->company_id);

        abort_unless($ownsOrder, 403, 'Not your order.');
    }
}
