<?php

use App\Http\Controllers\Api\AuthController;
use App\Http\Controllers\Api\CatalogMetaController;
use App\Http\Controllers\Api\OrderController;
use App\Http\Controllers\Api\PaymentController;
use App\Http\Controllers\Api\ProductController;
use App\Http\Controllers\Api\Admin;
use App\Http\Middleware\EnsureAdmin;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| Public storefront (B2C + browsing)
|--------------------------------------------------------------------------
*/
Route::post('/auth/register', [AuthController::class, 'register']);
Route::post('/auth/login', [AuthController::class, 'login']);

Route::get('/categories', [CatalogMetaController::class, 'categories']);
Route::get('/currencies', [CatalogMetaController::class, 'currencies']);
Route::get('/languages', [CatalogMetaController::class, 'languages']);
Route::get('/payment-gateways', [PaymentController::class, 'gateways']);

Route::get('/products', [ProductController::class, 'index']);
Route::get('/products/{idOrSlug}', [ProductController::class, 'show']);
Route::get('/products/{product}/analogs', [ProductController::class, 'analogs']);

// Provider webhooks (no auth; verified inside each gateway driver).
Route::post('/payments/{gateway}/callback', [PaymentController::class, 'callback']);

/*
|--------------------------------------------------------------------------
| Authenticated (B2C + B2B)
|--------------------------------------------------------------------------
*/
Route::middleware('auth:sanctum')->group(function () {
    Route::get('/auth/me', [AuthController::class, 'me']);
    Route::post('/auth/logout', [AuthController::class, 'logout']);

    Route::get('/orders', [OrderController::class, 'index']);
    Route::post('/orders', [OrderController::class, 'store']);
    Route::get('/orders/{order}', [OrderController::class, 'show']);

    Route::post('/orders/{order}/pay', [PaymentController::class, 'pay']);
});

/*
|--------------------------------------------------------------------------
| Admin panel (role: admin)
|--------------------------------------------------------------------------
*/
Route::middleware(['auth:sanctum', EnsureAdmin::class])->prefix('admin')->group(function () {
    Route::get('/dashboard', [Admin\DashboardController::class, 'index']);

    Route::get('/users', [Admin\UserController::class, 'index']);
    Route::post('/users', [Admin\UserController::class, 'store']);
    Route::put('/users/{user}', [Admin\UserController::class, 'update']);
    Route::delete('/users/{user}', [Admin\UserController::class, 'destroy']);

    Route::get('/orders', [Admin\OrderController::class, 'index']);
    Route::get('/orders/{order}', [Admin\OrderController::class, 'show']);
    Route::put('/orders/{order}/status', [Admin\OrderController::class, 'updateStatus']);

    Route::get('/products', [Admin\ProductController::class, 'index']);
    Route::post('/products', [Admin\ProductController::class, 'store']);
    Route::put('/products/{product}', [Admin\ProductController::class, 'update']);
    Route::delete('/products/{product}', [Admin\ProductController::class, 'destroy']);

    // Per-product price overrides & B2B volume tiers.
    Route::get('/products/{product}/prices', [Admin\ProductPriceController::class, 'index']);
    Route::post('/products/{product}/prices', [Admin\ProductPriceController::class, 'store']);
    Route::put('/products/{product}/prices/{price}', [Admin\ProductPriceController::class, 'update']);
    Route::delete('/products/{product}/prices/{price}', [Admin\ProductPriceController::class, 'destroy']);

    Route::get('/categories', [Admin\CategoryController::class, 'index']);
    Route::post('/categories', [Admin\CategoryController::class, 'store']);
    Route::put('/categories/{category}', [Admin\CategoryController::class, 'update']);
    Route::delete('/categories/{category}', [Admin\CategoryController::class, 'destroy']);

    Route::get('/currencies', [Admin\CurrencyController::class, 'index']);
    Route::post('/currencies', [Admin\CurrencyController::class, 'store']);
    Route::put('/currencies/{currency}', [Admin\CurrencyController::class, 'update']);
    Route::delete('/currencies/{currency}', [Admin\CurrencyController::class, 'destroy']);

    Route::get('/companies', [Admin\CompanyController::class, 'index']);
    Route::put('/companies/{company}', [Admin\CompanyController::class, 'update']);
});
