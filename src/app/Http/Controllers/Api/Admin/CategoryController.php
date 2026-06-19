<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Models\Category;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use Illuminate\Validation\Rule;

class CategoryController extends Controller
{
    public function index(): JsonResponse
    {
        return response()->json(
            Category::withCount('products')->orderBy('position')->orderBy('name')->get()
        );
    }

    public function store(Request $request): JsonResponse
    {
        $data = $this->validateData($request);
        $data['slug'] = $data['slug'] ?? Str::slug($data['name']);

        return response()->json(Category::create($data), 201);
    }

    public function update(Request $request, Category $category): JsonResponse
    {
        $category->update($this->validateData($request, $category->id));

        return response()->json($category->fresh());
    }

    public function destroy(Category $category): JsonResponse
    {
        $category->delete();

        return response()->json(['message' => 'Category deleted.']);
    }

    protected function validateData(Request $request, ?int $id = null): array
    {
        return $request->validate([
            'name' => [$id ? 'sometimes' : 'required', 'string', 'max:255'],
            'slug' => ['nullable', 'string', Rule::unique('categories', 'slug')->ignore($id)],
            'parent_id' => ['nullable', 'exists:categories,id'],
            'name_translations' => ['nullable', 'array'],
            'position' => ['nullable', 'integer'],
            'is_active' => ['boolean'],
        ]);
    }
}
