<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('companies', function (Blueprint $table) {
            $table->id();
            $table->string('name');
            $table->string('slug')->unique();
            $table->string('tax_number')->nullable();          // VAT / company registration
            $table->string('email')->nullable();
            $table->string('phone')->nullable();
            $table->string('country', 2)->nullable();          // ISO country code
            $table->text('address')->nullable();
            $table->string('default_currency', 3)->default('USD');
            $table->string('default_locale', 5)->default('en');
            $table->boolean('is_verified')->default(false);    // B2B accounts often need vetting
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('companies');
    }
};
