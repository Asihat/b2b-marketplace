<?php

namespace App\Http\Requests\Auth;

use App\Enums\AccountType;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class RegisterRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', 'unique:users,email'],
            'password' => ['required', 'string', 'min:8'],
            'type' => ['nullable', Rule::enum(AccountType::class)],
            'phone' => ['nullable', 'string', 'max:50'],
            'locale' => ['nullable', 'string', 'max:5'],
            'currency' => ['nullable', 'string', 'size:3'],
            'company_name' => ['required_if:type,b2b', 'string', 'max:255'],
            'company_tax_number' => ['nullable', 'string', 'max:100'],
            'company_country' => ['nullable', 'string', 'size:2'],
        ];
    }
}
