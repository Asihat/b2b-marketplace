<?php

namespace App\Services;

use App\Enums\AccountType;
use App\Enums\OrderStatus;
use App\Enums\PaymentStatus;
use App\Models\Order;
use App\Models\Payment;
use App\Models\Product;
use App\Models\User;
use App\Payments\PaymentManager;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;
use RuntimeException;

class OrderService
{
    public function __construct(
        protected CurrencyService $currency,
        protected PaymentManager $payments,
    ) {}

    /**
     * Create an order for a user from a list of [product_id, quantity] lines.
     *
     * @param  array<int, array{product_id:int, quantity:int}>  $lines
     */
    public function place(User $user, array $lines, array $attributes = []): Order
    {
        if (empty($lines)) {
            throw new RuntimeException('Cannot place an empty order.');
        }

        $currency = $attributes['currency_code'] ?? $user->currency ?? $this->currency->base()['code'];
        $taxRate = (float) ($attributes['tax_rate'] ?? 0); // simple flat tax; override per-region

        return DB::transaction(function () use ($user, $lines, $attributes, $currency, $taxRate) {
            $order = Order::create([
                'number' => 'ORD-'.strtoupper(Str::random(8)),
                'user_id' => $user->id,
                'company_id' => $user->company_id,
                'type' => $user->isB2b() ? AccountType::B2b : AccountType::B2c,
                'status' => OrderStatus::Pending,
                'currency_code' => $currency,
                'contact_name' => $attributes['contact_name'] ?? $user->name,
                'contact_email' => $attributes['contact_email'] ?? $user->email,
                'contact_phone' => $attributes['contact_phone'] ?? $user->phone,
                'shipping_address' => $attributes['shipping_address'] ?? null,
                'shipping_city' => $attributes['shipping_city'] ?? null,
                'shipping_postal_code' => $attributes['shipping_postal_code'] ?? null,
                'shipping_country' => $attributes['shipping_country'] ?? null,
                'notes' => $attributes['notes'] ?? null,
            ]);

            $subtotal = 0.0;

            foreach ($lines as $line) {
                /** @var Product $product */
                $product = Product::active()->lockForUpdate()->findOrFail($line['product_id']);
                $qty = max(1, (int) $line['quantity']);

                if ($product->is_b2b_only && ! $user->isB2b()) {
                    throw new RuntimeException("Product {$product->sku} is available to business accounts only.");
                }
                if ($qty < $product->min_order_qty) {
                    throw new RuntimeException("Minimum order quantity for {$product->sku} is {$product->min_order_qty}.");
                }
                if ($product->stock < $qty) {
                    throw new RuntimeException("Insufficient stock for {$product->sku}.");
                }

                $unitPrice = $this->currency->priceFor($product, $currency, $qty);
                $lineTotal = round($unitPrice * $qty, 4);
                $subtotal += $lineTotal;

                $order->items()->create([
                    'product_id' => $product->id,
                    'name' => $product->name,
                    'sku' => $product->sku,
                    'quantity' => $qty,
                    'unit_price' => $unitPrice,
                    'line_total' => $lineTotal,
                ]);

                $product->decrement('stock', $qty);
            }

            $tax = round($subtotal * $taxRate, 4);
            $order->update([
                'subtotal' => $subtotal,
                'tax_total' => $tax,
                'grand_total' => round($subtotal + $tax, 4),
            ]);

            return $order->load('items');
        });
    }

    /**
     * Charge an order through a gateway and record the payment.
     */
    public function pay(Order $order, ?string $gateway = null): Payment
    {
        $driver = $this->payments->driver($gateway);
        $result = $driver->charge($order);

        $completed = $result->status === PaymentStatus::Completed->value;

        $payment = $order->payments()->create([
            'gateway' => $driver->name(),
            'status' => $result->status,
            'currency_code' => $order->currency_code,
            'amount' => $order->grand_total,
            'reference' => $result->reference,
            'payload' => $result->payload,
            'paid_at' => $completed ? now() : null,
        ]);

        if ($completed) {
            $order->update(['status' => OrderStatus::Paid]);
        }

        return $payment;
    }
}
