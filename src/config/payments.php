<?php

return [
    /*
     | Default payment gateway used when an order does not specify one.
     | Built-in drivers: "fake" (sandbox), "manual" (bank transfer).
     | Add your own by implementing App\Contracts\PaymentGateway and
     | registering it in App\Payments\PaymentManager::$drivers.
     */
    'default' => env('PAYMENT_GATEWAY', 'fake'),

    /*
     | Per-gateway credentials/options. Read by each driver as needed.
     */
    'gateways' => [
        'fake' => [],
        'manual' => [
            'bank_account' => env('PAYMENT_MANUAL_BANK_ACCOUNT', ''),
        ],
        // 'stripe' => [
        //     'secret' => env('STRIPE_SECRET'),
        //     'webhook_secret' => env('STRIPE_WEBHOOK_SECRET'),
        // ],
    ],
];
