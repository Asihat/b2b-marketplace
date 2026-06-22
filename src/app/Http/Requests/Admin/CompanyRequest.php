<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;

class CompanyRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'name' => ['sometimes', 'string', 'max:255'],
            'tax_number' => ['nullable', 'string', 'max:100'],
            'email' => ['nullable', 'email'],
            'phone' => ['nullable', 'string', 'max:50'],
            'country' => ['nullable', 'string', 'size:2'],
            'address' => ['nullable', 'string'],
            'default_currency' => ['nullable', 'string', 'size:3'],
            'default_locale' => ['nullable', 'string', 'max:5'],
            'is_verified' => ['boolean'],
        ];
    }
}
