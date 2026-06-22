<?php

namespace App\Services;

use App\Enums\AccountType;
use App\Enums\UserRole;
use App\Models\Company;
use App\Models\User;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;

class AccountService
{
    /**
     * Register a retail (B2C) or business (B2B) account. B2B registrations
     * also create the associated company.
     *
     * @param  array<string, mixed>  $data  Validated registration payload.
     */
    public function register(array $data): User
    {
        return DB::transaction(function () use ($data) {
            $type = AccountType::from($data['type'] ?? AccountType::B2c->value);

            $company = $type === AccountType::B2b
                ? $this->createCompany($data)
                : null;

            return User::create([
                'company_id' => $company?->id,
                'name' => $data['name'],
                'email' => $data['email'],
                'password' => $data['password'],
                'type' => $type,
                'role' => UserRole::Customer,
                'phone' => $data['phone'] ?? null,
                'locale' => $data['locale'] ?? 'en',
                'currency' => $data['currency'] ?? 'USD',
            ]);
        });
    }

    /** @param array<string, mixed> $data */
    private function createCompany(array $data): Company
    {
        return Company::create([
            'name' => $data['company_name'],
            'slug' => Str::slug($data['company_name']).'-'.Str::lower(Str::random(5)),
            'tax_number' => $data['company_tax_number'] ?? null,
            'country' => $data['company_country'] ?? null,
            'default_currency' => $data['currency'] ?? 'USD',
            'default_locale' => $data['locale'] ?? 'en',
        ]);
    }
}
