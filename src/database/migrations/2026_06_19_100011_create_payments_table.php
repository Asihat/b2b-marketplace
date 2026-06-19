<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('payments', function (Blueprint $table) {
            $table->id();
            $table->foreignId('order_id')->constrained()->cascadeOnDelete();
            $table->string('gateway');                     // fake | manual | stripe | ...
            $table->string('status')->default('pending');  // pending|completed|failed|refunded
            $table->string('currency_code', 3);
            $table->decimal('amount', 18, 4);
            $table->string('reference')->nullable();        // gateway transaction id
            $table->json('payload')->nullable();            // raw gateway response
            $table->timestamp('paid_at')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('payments');
    }
};
