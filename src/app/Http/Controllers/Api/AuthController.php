<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Company;
use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Str;
use Illuminate\Validation\ValidationException;

class AuthController extends Controller
{
    /**
     * Register a B2C customer or a B2B account (optionally creating a company).
     */
    public function register(Request $request): JsonResponse
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', 'unique:users,email'],
            'password' => ['required', 'string', 'min:8'],
            'type' => ['nullable', 'in:b2c,b2b'],
            'phone' => ['nullable', 'string', 'max:50'],
            'locale' => ['nullable', 'string', 'max:5'],
            'currency' => ['nullable', 'string', 'size:3'],
            // B2B: supply company details to create/attach a company.
            'company_name' => ['required_if:type,b2b', 'string', 'max:255'],
            'company_tax_number' => ['nullable', 'string', 'max:100'],
            'company_country' => ['nullable', 'string', 'size:2'],
        ]);

        $type = $data['type'] ?? 'b2c';
        $companyId = null;

        if ($type === 'b2b') {
            $company = Company::create([
                'name' => $data['company_name'],
                'slug' => Str::slug($data['company_name']).'-'.Str::lower(Str::random(5)),
                'tax_number' => $data['company_tax_number'] ?? null,
                'country' => $data['company_country'] ?? null,
                'default_currency' => $data['currency'] ?? 'USD',
                'default_locale' => $data['locale'] ?? 'en',
            ]);
            $companyId = $company->id;
        }

        $user = User::create([
            'company_id' => $companyId,
            'name' => $data['name'],
            'email' => $data['email'],
            'password' => $data['password'],
            'type' => $type,
            'role' => 'customer',
            'phone' => $data['phone'] ?? null,
            'locale' => $data['locale'] ?? 'en',
            'currency' => $data['currency'] ?? 'USD',
        ]);

        $token = $user->createToken('api')->plainTextToken;

        return response()->json([
            'user' => $user->load('company'),
            'token' => $token,
        ], 201);
    }

    public function login(Request $request): JsonResponse
    {
        $data = $request->validate([
            'email' => ['required', 'email'],
            'password' => ['required', 'string'],
        ]);

        $user = User::where('email', $data['email'])->first();

        if (! $user || ! Hash::check($data['password'], $user->password)) {
            throw ValidationException::withMessages([
                'email' => ['The provided credentials are incorrect.'],
            ]);
        }

        if (! $user->is_active) {
            throw ValidationException::withMessages([
                'email' => ['This account is disabled.'],
            ]);
        }

        $token = $user->createToken('api')->plainTextToken;

        return response()->json([
            'user' => $user->load('company'),
            'token' => $token,
        ]);
    }

    public function me(Request $request): JsonResponse
    {
        return response()->json($request->user()->load('company'));
    }

    public function logout(Request $request): JsonResponse
    {
        $request->user()->currentAccessToken()->delete();

        return response()->json(['message' => 'Logged out.']);
    }
}
