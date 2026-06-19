<?php

namespace App\Payments\Gateways;

use App\Contracts\PaymentGateway;
use App\Models\Order;
use App\Payments\PaymentResult;
use Illuminate\Support\Str;

/**
 * Development/sandbox gateway. Approves every charge instantly so the full
 * order -> payment -> fulfillment flow can be exercised without a real provider.
 */
class FakeGateway implements PaymentGateway
{
    public function name(): string
    {
        return 'fake';
    }

    public function charge(Order $order, array $options = []): PaymentResult
    {
        return PaymentResult::completed(
            reference: 'fake_'.Str::uuid(),
            payload: ['simulated' => true, 'order' => $order->number],
        );
    }

    public function callback(array $payload): PaymentResult
    {
        return PaymentResult::completed($payload['reference'] ?? 'fake_callback', $payload);
    }

    public function refund(string $reference, float $amount): PaymentResult
    {
        return PaymentResult::completed($reference, ['refunded' => $amount]);
    }
}
