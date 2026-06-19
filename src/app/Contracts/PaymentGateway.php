<?php

namespace App\Contracts;

use App\Models\Order;
use App\Payments\PaymentResult;

/**
 * Contract every payment provider must implement.
 *
 * Drop in a new gateway (Stripe, PayPal, a local bank, crypto, ...) by
 * implementing this interface and registering it in PaymentManager.
 */
interface PaymentGateway
{
    /** Machine name, e.g. "stripe". Must match config/env selection. */
    public function name(): string;

    /** Begin a charge for the given order. */
    public function charge(Order $order, array $options = []): PaymentResult;

    /**
     * Handle an asynchronous provider callback / webhook.
     * Returns the resolved result for the referenced payment.
     */
    public function callback(array $payload): PaymentResult;

    /** Refund a previously completed payment by gateway reference. */
    public function refund(string $reference, float $amount): PaymentResult;
}
