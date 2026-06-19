<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('products', function (Blueprint $table) {
            $table->id();
            $table->foreignId('category_id')->nullable()->constrained()->nullOnDelete();
            // Seller company (B2B sellers). Null = marketplace-owned / B2C catalog item.
            $table->foreignId('company_id')->nullable()->constrained()->nullOnDelete();
            $table->string('sku')->unique();
            $table->string('slug')->unique();
            $table->string('name');                 // default-locale name
            $table->text('description')->nullable();
            $table->string('brand')->nullable();
            $table->string('unit')->default('pcs'); // pcs, kg, box...
            $table->decimal('base_price', 18, 4)->default(0); // price in base currency
            $table->unsignedInteger('stock')->default(0);
            $table->unsignedInteger('min_order_qty')->default(1); // B2B MOQ
            $table->boolean('is_b2b_only')->default(false);
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index('brand');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('products');
    }
};
