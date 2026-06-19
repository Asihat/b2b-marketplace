<?php

namespace App\Payments\Gateways;

use App\Contracts\PaymentGateway;
use App\Models\Order;
use App\Payments\PaymentResult;
use Illuminate\Support\Str;

/**
 * Offline / bank-transfer gateway, common for B2B. The charge stays "pending"
 * until an operator confirms the incoming wire via the callback/confirm action.
 */
class ManualGateway implements PaymentGateway
{
    public function name(): string
    {
        return 'manual';
    }

    public function charge(Order $order, array $options = []): PaymentResult
    {
        return PaymentResult::pending(
            reference: 'manual_'.Str::upper(Str::random(10)),
            payload: [
                'instructions' => 'Transfer the order total to the marketplace bank account using the order number as reference.',
                'order' => $order->number,
            ],
        );
    }

    public function callback(array $payload): PaymentResult
    {
        // An operator confirms receipt of funds.
        return PaymentResult::completed($payload['reference'] ?? 'manual_confirmed', $payload);
    }

    public function refund(string $reference, float $amount): PaymentResult
    {
        return PaymentResult::pending($reference, payload: ['refund_requested' => $amount]);
    }
}
