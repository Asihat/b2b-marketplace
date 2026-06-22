<?php

namespace App\Http\Requests\Admin;

use App\Enums\AccountType;
use App\Enums\UserRole;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class UserRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $id = $this->route('user')?->id;
        $creating = $this->isMethod('post');

        return [
            'name' => [$creating ? 'required' : 'sometimes', 'string', 'max:255'],
            'email' => [$creating ? 'required' : 'sometimes', 'email', Rule::unique('users', 'email')->ignore($id)],
            'password' => [$creating ? 'required' : 'nullable', 'string', 'min:8'],
            'type' => [$creating ? 'required' : 'sometimes', Rule::enum(AccountType::class)],
            'role' => [$creating ? 'required' : 'sometimes', Rule::enum(UserRole::class)],
            'company_id' => ['nullable', 'exists:companies,id'],
            'currency' => ['nullable', 'string', 'size:3'],
            'locale' => ['nullable', 'string', 'max:5'],
            'is_active' => ['boolean'],
        ];
    }
}
