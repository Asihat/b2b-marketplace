<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Order;
use App\Payments\PaymentManager;
use App\Services\OrderService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PaymentController extends Controller
{
    public function __construct(
        protected OrderService $orders,
        protected PaymentManager $gateways,
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
        abort_unless(
            $order->user_id === $request->user()->id
                || ($request->user()->isB2b() && $order->company_id === $request->user()->company_id),
            403,
            'Not your order.'
        );

        if (in_array($order->status, ['paid', 'completed'], true)) {
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

        if ($result->reference) {
            $payment = \App\Models\Payment::where('reference', $result->reference)->first();
            if ($payment) {
                $payment->update([
                    'status' => $result->status,
                    'paid_at' => $result->status === 'completed' ? now() : $payment->paid_at,
                    'payload' => $result->payload,
                ]);
                if ($result->status === 'completed') {
                    $payment->order->update(['status' => 'paid']);
                }
            }
        }

        return response()->json(['status' => $result->status]);
    }
}
