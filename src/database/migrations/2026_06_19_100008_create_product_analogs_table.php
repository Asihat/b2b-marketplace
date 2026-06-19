<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        // Analogs / cross-references: interchangeable or substitute products.
        // Stored as a directed pair; the seeder/service keeps it symmetric.
        Schema::create('product_analogs', function (Blueprint $table) {
            $table->id();
            $table->foreignId('product_id')->constrained()->cascadeOnDelete();
            $table->foreignId('analog_id')->constrained('products')->cascadeOnDelete();
            $table->string('type')->default('equivalent'); // equivalent | substitute | upgrade
            $table->text('note')->nullable();
            $table->timestamps();

            $table->unique(['product_id', 'analog_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('product_analogs');
    }
};
