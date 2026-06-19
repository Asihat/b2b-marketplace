<?php

namespace App\Payments;

use App\Contracts\PaymentGateway;
use App\Payments\Gateways\FakeGateway;
use App\Payments\Gateways\ManualGateway;
use InvalidArgumentException;

/**
 * Resolves a payment gateway driver by name. Register additional drivers
 * (Stripe, PayPal, ...) in the $drivers map or via extend().
 */
class PaymentManager
{
    /** @var array<string, class-string<PaymentGateway>> */
    protected array $drivers = [
        'fake' => FakeGateway::class,
        'manual' => ManualGateway::class,
    ];

    /** @var array<string, PaymentGateway> */
    protected array $resolved = [];

    public function driver(?string $name = null): PaymentGateway
    {
        $name ??= config('payments.default', 'fake');

        if (! isset($this->drivers[$name])) {
            throw new InvalidArgumentException("Payment gateway [$name] is not configured.");
        }

        return $this->resolved[$name] ??= app($this->drivers[$name]);
    }

    /** Register a custom gateway at runtime (e.g. from a package). */
    public function extend(string $name, string $class): void
    {
        $this->drivers[$name] = $class;
    }

    /** @return array<int, string> */
    public function available(): array
    {
        return array_keys($this->drivers);
    }
}
