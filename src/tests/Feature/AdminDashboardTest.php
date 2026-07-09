<?php

namespace Tests\Feature;

use App\Enums\OrderStatus;
use App\Enums\PaymentStatus;
use App\Enums\UserRole;
use App\Models\Category;
use App\Models\Currency;
use App\Models\Order;
use App\Models\OrderItem;
use App\Models\Payment;
use App\Models\Product;
use App\Models\User;
use App\Services\CurrencyService;
use Carbon\CarbonImmutable;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Sanctum\Sanctum;
use Tests\TestCase;

class AdminDashboardTest extends TestCase
{
    use RefreshDatabase;

    private User $buyer;

    protected function setUp(): void
    {
        parent::setUp();

        Sanctum::actingAs(User::factory()->create(['role' => UserRole::Admin]));
        $this->buyer = User::factory()->create(['role' => UserRole::Customer, 'type' => 'b2c']);

        Currency::updateOrCreate(['code' => 'USD'], ['name' => 'US Dollar', 'symbol' => '$', 'exchange_rate' => 1.0, 'is_base' => true]);
        Currency::updateOrCreate(['code' => 'EUR'], ['name' => 'Euro', 'symbol' => '€', 'exchange_rate' => 0.5, 'is_base' => false]);

        // Rates are cached for an hour; the fixtures above must be the ones used.
        app(CurrencyService::class)->flush();
    }

    public function test_revenue_is_converted_into_the_base_currency_and_compared_to_the_previous_window(): void
    {
        // Current 30-day window: 100 EUR (= 200 USD at 0.5) + 50 USD = 250 USD.
        $this->order(['currency_code' => 'EUR', 'grand_total' => 100, 'status' => OrderStatus::Completed], daysAgo: 2);
        $this->order(['currency_code' => 'USD', 'grand_total' => 50, 'status' => OrderStatus::Paid], daysAgo: 1);
        // Pending never counts as revenue, but it is still an order.
        $this->order(['currency_code' => 'USD', 'grand_total' => 999, 'status' => OrderStatus::Pending], daysAgo: 3);
        // Previous window (days 31–60 ago).
        $this->order(['currency_code' => 'USD', 'grand_total' => 125, 'status' => OrderStatus::Completed], daysAgo: 40);
        // Outside both windows entirely.
        $this->order(['currency_code' => 'USD', 'grand_total' => 5000, 'status' => OrderStatus::Completed], daysAgo: 200);

        $response = $this->getJson('/api/admin/dashboard?days=30')->assertOk();

        $response->assertJsonPath('base_currency.code', 'USD')
            ->assertJsonPath('kpis.orders.current', 3);

        // Whole amounts serialize as JSON integers, so compare numerically.
        $this->assertMoney(250, $response->json('kpis.revenue.current'));
        $this->assertMoney(125, $response->json('kpis.revenue.previous'));
        $this->assertMoney(100, $response->json('kpis.revenue.change'));
        // Average is over the two revenue-bearing orders only, not the pending one.
        $this->assertMoney(125, $response->json('kpis.aov.current'));
    }

    public function test_timeseries_covers_every_day_of_the_range(): void
    {
        $this->order(['currency_code' => 'EUR', 'grand_total' => 10, 'status' => OrderStatus::Completed], daysAgo: 1);

        $response = $this->getJson('/api/admin/dashboard?days=7')->assertOk();

        $timeseries = $response->json('timeseries');
        $yesterday = CarbonImmutable::now()->subDays(1)->toDateString();

        $this->assertCount(7, $timeseries);
        $this->assertMoney(20, collect($timeseries)->firstWhere('date', $yesterday)['revenue']);
        $this->assertSame(1, array_sum(array_column($timeseries, 'orders')));
    }

    public function test_leaderboards_and_gateways_report_base_currency_totals(): void
    {
        $category = Category::create(['slug' => 'tools', 'name' => 'Tools', 'position' => 0]);
        $product = Product::create([
            'category_id' => $category->id, 'sku' => 'W-1', 'slug' => 'wrench',
            'name' => 'Wrench', 'base_price' => 20, 'stock' => 10,
        ]);

        $order = $this->order(['currency_code' => 'EUR', 'grand_total' => 40, 'status' => OrderStatus::Completed], daysAgo: 1);
        (new OrderItem)->forceFill([
            'order_id' => $order->id, 'product_id' => $product->id, 'name' => 'Wrench',
            'sku' => 'W-1', 'quantity' => 2, 'unit_price' => 20, 'line_total' => 40,
        ])->save();

        (new Payment)->forceFill([
            'order_id' => $order->id, 'gateway' => 'fake', 'status' => PaymentStatus::Completed,
            'currency_code' => 'EUR', 'amount' => 40, 'paid_at' => now(),
        ])->save();

        $response = $this->getJson('/api/admin/dashboard?days=30')->assertOk();

        $response->assertJsonPath('top_products.0.sku', 'W-1')
            ->assertJsonPath('top_products.0.qty', 2)
            ->assertJsonPath('top_categories.0.name', 'Tools')
            ->assertJsonPath('gateways.0.gateway', 'fake');

        // 40 EUR at a 0.5 rate is 80 USD everywhere it is reported.
        $this->assertMoney(80, $response->json('top_products.0.revenue'));
        $this->assertMoney(80, $response->json('top_categories.0.revenue'));
        $this->assertMoney(80, $response->json('gateways.0.amount'));
    }

    public function test_order_status_breakdown_lists_every_status(): void
    {
        $this->order(['currency_code' => 'USD', 'grand_total' => 10, 'status' => OrderStatus::Cancelled], daysAgo: 1);

        $breakdown = collect($this->getJson('/api/admin/dashboard')->assertOk()->json('orders_by_status'))
            ->pluck('count', 'status');

        $this->assertCount(count(OrderStatus::cases()), $breakdown);
        $this->assertSame(1, $breakdown['cancelled']);
        $this->assertSame(0, $breakdown['completed']);
    }

    public function test_the_range_is_restricted_to_supported_windows(): void
    {
        $this->getJson('/api/admin/dashboard?days=365')->assertJsonValidationErrors('days');
        $this->getJson('/api/admin/dashboard?days=7')->assertOk()->assertJsonPath('range.days', 7);
        $this->getJson('/api/admin/dashboard')->assertOk()->assertJsonPath('range.days', 30);
    }

    public function test_the_dashboard_is_closed_to_non_admins(): void
    {
        Sanctum::actingAs($this->buyer);

        $this->getJson('/api/admin/dashboard')->assertForbidden();
    }

    /** JSON encodes 80.0 as 80, so money assertions compare value, not type. */
    private function assertMoney(float $expected, mixed $actual): void
    {
        $this->assertEqualsWithDelta($expected, $actual, 0.001);
    }

    private function order(array $attributes, int $daysAgo): Order
    {
        $placedAt = CarbonImmutable::now()->subDays($daysAgo)->setTime(12, 0);

        $order = new Order;
        $order->forceFill([
            'number' => 'ORD-'.str()->random(8),
            'user_id' => $this->buyer->id,
            'type' => 'b2c',
            'subtotal' => $attributes['grand_total'],
            'created_at' => $placedAt,
            'updated_at' => $placedAt,
            ...$attributes,
        ])->save();

        return $order;
    }
}
