<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('users', function (Blueprint $table) {
            // B2B buyers/sellers belong to a company; B2C users do not.
            $table->foreignId('company_id')->nullable()->after('id')->constrained()->nullOndelete();
            $table->string('type')->default('b2c')->after('email');   // b2c | b2b
            $table->string('role')->default('customer')->after('type'); // customer | manager | admin
            $table->string('phone')->nullable()->after('role');
            $table->string('locale', 5)->default('en')->after('phone');
            $table->string('currency', 3)->default('USD')->after('locale');
            $table->boolean('is_active')->default(true)->after('currency');
        });
    }

    public function down(): void
    {
        Schema::table('users', function (Blueprint $table) {
            $table->dropConstrainedForeignId('company_id');
            $table->dropColumn(['type', 'role', 'phone', 'locale', 'currency', 'is_active']);
        });
    }
};
