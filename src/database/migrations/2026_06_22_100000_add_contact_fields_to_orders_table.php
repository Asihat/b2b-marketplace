<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('orders', function (Blueprint $table) {
            $table->string('contact_name')->nullable()->after('type');
            $table->string('contact_email')->nullable()->after('contact_name');
            $table->string('contact_phone')->nullable()->after('contact_email');
            $table->string('shipping_city')->nullable()->after('shipping_address');
            $table->string('shipping_postal_code')->nullable()->after('shipping_city');
            $table->string('shipping_country', 2)->nullable()->after('shipping_postal_code');
        });
    }

    public function down(): void
    {
        Schema::table('orders', function (Blueprint $table) {
            $table->dropColumn([
                'contact_name', 'contact_email', 'contact_phone',
                'shipping_city', 'shipping_postal_code', 'shipping_country',
            ]);
        });
    }
};
