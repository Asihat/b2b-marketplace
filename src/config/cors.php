<?php

return [
    // Token-based (Bearer) API consumed by the SPA storefront.
    'paths' => ['api/*', 'up'],

    'allowed_methods' => ['*'],

    // Dev-friendly: allow any origin. Lock this down to your storefront
    // domain(s) in production.
    'allowed_origins' => ['*'],

    'allowed_origins_patterns' => [],

    'allowed_headers' => ['*'],

    'exposed_headers' => [],

    'max_age' => 0,

    // Bearer tokens, not cookies -> credentials not required.
    'supports_credentials' => false,
];
