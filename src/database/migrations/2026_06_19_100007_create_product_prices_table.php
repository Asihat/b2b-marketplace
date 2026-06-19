<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        // Optional per-currency price overrides and B2B tiered (volume) pricing.
        // If no row exists for a currency, price is derived from base_price * exchange_rate.
        Schema::create('product_prices', function (Blueprint $table) {
            $table->id();
            $table->foreignId('product_id')->constrained()->cascadeOnDelete();
            $table->string('currency_code', 3);
            $table->unsignedInteger('min_qty')->default(1); // tier threshold for B2B
            $table->decimal('price', 18, 4);
            $table->timestamps();

            $table->unique(['product_id', 'currency_code', 'min_qty']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('product_prices');
    }
};
