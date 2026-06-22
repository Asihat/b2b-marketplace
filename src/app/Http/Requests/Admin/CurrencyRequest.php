<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class CurrencyRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $id = $this->route('currency')?->id;
        $required = $this->isMethod('post') ? 'required' : 'sometimes';

        return [
            'code' => [$required, 'string', 'size:3', Rule::unique('currencies', 'code')->ignore($id)],
            'name' => [$required, 'string', 'max:255'],
            'symbol' => [$required, 'string', 'max:8'],
            'exchange_rate' => [$required, 'numeric', 'min:0'],
            'is_base' => ['boolean'],
            'is_active' => ['boolean'],
        ];
    }
}
