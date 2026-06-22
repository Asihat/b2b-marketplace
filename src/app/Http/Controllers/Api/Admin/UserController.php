<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\UserRequest;
use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class UserController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $users = User::query()
            ->with('company:id,name')
            ->search($request->query('search'))
            ->when($request->query('type'), fn ($q, $t) => $q->where('type', $t))
            ->when($request->query('role'), fn ($q, $r) => $q->where('role', $r))
            ->latest()
            ->paginate(20);

        return response()->json($users);
    }

    public function store(UserRequest $request): JsonResponse
    {
        $user = User::create($request->validated());

        return response()->json($user->load('company:id,name'), 201);
    }

    public function update(UserRequest $request, User $user): JsonResponse
    {
        $data = collect($request->validated())
            ->reject(fn ($value, $key) => $key === 'password' && blank($value))
            ->all();

        $user->update($data);

        return response()->json($user->fresh()->load('company:id,name'));
    }

    public function destroy(Request $request, User $user): JsonResponse
    {
        abort_if($user->id === $request->user()->id, 422, 'You cannot delete your own account.');

        $user->delete();

        return response()->json(['message' => 'User deleted.']);
    }
}
