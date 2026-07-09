<?php

namespace Database\Seeders;

use App\Enums\AccountType;
use App\Enums\OrderStatus;
use App\Enums\PaymentStatus;
use App\Models\Order;
use App\Models\OrderItem;
use App\Models\Payment;
use App\Models\Product;
use App\Models\User;
use App\Services\CurrencyService;
use Carbon\CarbonImmutable;
use Illuminate\Database\Seeder;
use Illuminate\Support\Collection;
use Illuminate\Support\Str;

/**
 * Back-dated demo orders so the admin dashboard has trends, leaderboards and
 * revenue to show on a fresh install.
 *
 * Orders are written directly (not through OrderService) because they need
 * historical timestamps and must not decrement today's stock levels.
 */
class DemoOrderSeeder extends Seeder
{
    /** How far back the demo history reaches — matches the dashboard's widest range. */
    private const DAYS = 90;

    private const ORDER_COUNT = 140;

    /** Status → weight. Roughly mirrors a healthy funnel with some churn. */
    private const STATUS_WEIGHTS = [
        'completed' => 34,
        'shipped' => 15,
        'processing' => 12,
        'paid' => 14,
        'pending' => 15,
        'cancelled' => 10,
    ];

    private const GATEWAY_WEIGHTS = ['fake' => 70, 'manual' => 30];

    public function __construct(private readonly CurrencyService $currency) {}

    public function run(): void
    {
        if (Order::exists()) {
            $this->command?->warn('Orders already exist — skipping demo order seeding.');

            return;
        }

        $buyers = User::where('role', '!=', 'admin')->get();
        $products = Product::where('is_active', true)->get();

        if ($buyers->isEmpty() || $products->isEmpty()) {
            $this->command?->warn('No buyers or products to build demo orders from.');

            return;
        }

        // Deterministic history: re-seeding produces the same demo numbers.
        mt_srand(20260101);

        $today = CarbonImmutable::now();

        for ($i = 0; $i < self::ORDER_COUNT; $i++) {
            $placedAt = $this->randomOrderDate($today);
            $this->createOrder($buyers->random(), $products, $placedAt);
        }

        mt_srand();

        $this->command?->info(self::ORDER_COUNT.' demo orders seeded across the last '.self::DAYS.' days.');
    }

    /**
     * A random moment in the window, biased towards recent days so the
     * dashboard's period-over-period deltas show growth rather than noise.
     */
    private function randomOrderDate(CarbonImmutable $today): CarbonImmutable
    {
        do {
            $daysAgo = mt_rand(0, self::DAYS - 1);
            $recency = 1 - ($daysAgo / self::DAYS);          // 0 = oldest, 1 = today
        } while (mt_rand(0, 100) > 35 + (65 * $recency));

        return $today->subDays($daysAgo)
            ->setTime(mt_rand(8, 20), mt_rand(0, 59), mt_rand(0, 59));
    }

    private function createOrder(User $buyer, Collection $products, CarbonImmutable $placedAt): void
    {
        $isB2b = $buyer->isB2b();
        $currency = $buyer->currency ?: $this->currency->base()['code'];
        $status = OrderStatus::from($this->weightedPick(self::STATUS_WEIGHTS));

        // B2C shoppers never see B2B-only stock, and buy in smaller baskets.
        $catalog = $isB2b ? $products : $products->where('is_b2b_only', false);
        $lines = $catalog->random(min($catalog->count(), mt_rand(1, $isB2b ? 4 : 3)));

        $order = new Order;
        $order->forceFill([
            'number' => 'ORD-'.strtoupper(Str::random(8)),
            'user_id' => $buyer->id,
            'company_id' => $buyer->company_id,
            'type' => $isB2b ? AccountType::B2b : AccountType::B2c,
            'status' => $status,
            'currency_code' => $currency,
            'contact_name' => $buyer->name,
            'contact_email' => $buyer->email,
            'created_at' => $placedAt,
            'updated_at' => $placedAt,
        ])->save();

        $subtotal = 0.0;

        foreach ($lines as $product) {
            $qty = $isB2b
                ? max($product->min_order_qty, mt_rand(1, 12) * $product->min_order_qty)
                : max($product->min_order_qty, mt_rand(1, 3));

            $unitPrice = $this->currency->priceFor($product, $currency, $qty);
            $lineTotal = round($unitPrice * $qty, 4);
            $subtotal += $lineTotal;

            (new OrderItem)->forceFill([
                'order_id' => $order->id,
                'product_id' => $product->id,
                'name' => $product->name,
                'sku' => $product->sku,
                'quantity' => $qty,
                'unit_price' => $unitPrice,
                'line_total' => $lineTotal,
                'created_at' => $placedAt,
                'updated_at' => $placedAt,
            ])->save();
        }

        $order->forceFill([
            'subtotal' => $subtotal,
            'tax_total' => 0,
            'grand_total' => $subtotal,
            'updated_at' => $placedAt,
        ])->save();

        if ($status !== OrderStatus::Pending && $status !== OrderStatus::Cancelled) {
            $this->createPayment($order, $placedAt);
        }
    }

    /** Settled orders get a completed payment shortly after they were placed. */
    private function createPayment(Order $order, CarbonImmutable $placedAt): void
    {
        $paidAt = $placedAt->addMinutes(mt_rand(1, 240));

        (new Payment)->forceFill([
            'order_id' => $order->id,
            'gateway' => $this->weightedPick(self::GATEWAY_WEIGHTS),
            'status' => PaymentStatus::Completed,
            'currency_code' => $order->currency_code,
            'amount' => $order->grand_total,
            'reference' => 'DEMO-'.strtoupper(Str::random(10)),
            'paid_at' => $paidAt,
            'created_at' => $paidAt,
            'updated_at' => $paidAt,
        ])->save();
    }

    /** @param array<string, int> $weights */
    private function weightedPick(array $weights): string
    {
        $roll = mt_rand(1, array_sum($weights));

        foreach ($weights as $key => $weight) {
            $roll -= $weight;
            if ($roll <= 0) {
                return $key;
            }
        }

        return array_key_first($weights);
    }
}
