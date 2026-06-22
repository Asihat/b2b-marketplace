<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\CategoryRequest;
use App\Models\Category;
use Illuminate\Http\JsonResponse;
use Illuminate\Support\Str;

class CategoryController extends Controller
{
    public function index(): JsonResponse
    {
        return response()->json(
            Category::withCount('products')->orderBy('position')->orderBy('name')->get()
        );
    }

    public function store(CategoryRequest $request): JsonResponse
    {
        $data = $request->validated();
        $data['slug'] ??= Str::slug($data['name']);

        return response()->json(Category::create($data), 201);
    }

    public function update(CategoryRequest $request, Category $category): JsonResponse
    {
        $category->update($request->validated());

        return response()->json($category->fresh());
    }

    public function destroy(Category $category): JsonResponse
    {
        $category->delete();

        return response()->json(['message' => 'Category deleted.']);
    }
}
