<?php

use App\Http\Controllers\ImageController;
use Illuminate\Support\Facades\Route;

Route::get('/', function () {
    return view('welcome');
});

// Locally-generated placeholder product images (no external dependency).
Route::get('/img/{seed}', [ImageController::class, 'placeholder'])->name('placeholder');
