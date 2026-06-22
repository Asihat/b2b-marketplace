<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use Illuminate\Http\Response;

/**
 * Generates deterministic SVG placeholder images, served by the app itself so
 * the storefront never depends on an external image host. The look (gradient +
 * initials) is derived from the {seed}, so each product keeps stable artwork.
 */
class ImageController extends Controller
{
    public function placeholder(Request $request, string $seed): Response
    {
        $hash = crc32($seed);
        $h1 = $hash % 360;
        $h2 = ($h1 + 45) % 360;

        // Derive 1–2 initials from the seed's words.
        $words = preg_split('/[^a-zA-Z0-9]+/', $seed, -1, PREG_SPLIT_NO_EMPTY) ?: ['?'];
        $initials = strtoupper(substr($words[0], 0, 1) . (isset($words[1]) ? substr($words[1], 0, 1) : ''));

        $svg = <<<SVG
        <svg xmlns="http://www.w3.org/2000/svg" width="800" height="800" viewBox="0 0 800 800">
          <defs>
            <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stop-color="hsl({$h1}, 62%, 56%)"/>
              <stop offset="1" stop-color="hsl({$h2}, 68%, 44%)"/>
            </linearGradient>
          </defs>
          <rect width="800" height="800" fill="url(#g)"/>
          <circle cx="640" cy="160" r="220" fill="#ffffff" opacity="0.08"/>
          <circle cx="160" cy="660" r="160" fill="#ffffff" opacity="0.06"/>
          <text x="50%" y="50%" dy="0.35em" text-anchor="middle"
            font-family="Inter, Segoe UI, Arial, sans-serif" font-size="300" font-weight="700"
            fill="#ffffff" opacity="0.92">{$initials}</text>
        </svg>
        SVG;

        return response($svg, 200)
            ->header('Content-Type', 'image/svg+xml')
            ->header('Cache-Control', 'public, max-age=31536000, immutable');
    }
}
