<?php

namespace App\Payments;

/**
 * Immutable result returned by every payment gateway driver.
 */
class PaymentResult
{
    public function __construct(
        public readonly bool $successful,
        public readonly string $status,        // completed | pending | failed
        public readonly ?string $reference = null,
        public readonly array $payload = [],
        public readonly ?string $redirectUrl = null, // for hosted/redirect gateways
        public readonly ?string $message = null,
    ) {}

    public static function completed(string $reference, array $payload = []): self
    {
        return new self(true, 'completed', $reference, $payload);
    }

    public static function pending(?string $reference = null, ?string $redirectUrl = null, array $payload = []): self
    {
        return new self(true, 'pending', $reference, $payload, $redirectUrl);
    }

    public static function failed(string $message, array $payload = []): self
    {
        return new self(false, 'failed', null, $payload, null, $message);
    }
}
