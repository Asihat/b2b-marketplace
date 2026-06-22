<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class CategoryRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $id = $this->route('category')?->id;

        return [
            'name' => [$this->isMethod('post') ? 'required' : 'sometimes', 'string', 'max:255'],
            'slug' => ['nullable', 'string', Rule::unique('categories', 'slug')->ignore($id)],
            'parent_id' => ['nullable', 'exists:categories,id'],
            'name_translations' => ['nullable', 'array'],
            'position' => ['nullable', 'integer'],
            'is_active' => ['boolean'],
        ];
    }
}
