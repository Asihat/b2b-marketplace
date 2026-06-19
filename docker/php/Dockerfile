# FrankenPHP ships PHP + the frankenphp binary used by Laravel Octane.
FROM dunglas/frankenphp:1-php8.4-alpine

# install-php-extensions is bundled in the FrankenPHP image.
RUN install-php-extensions \
        pdo_pgsql \
        pgsql \
        intl \
        bcmath \
        zip \
        opcache \
        pcntl \
        posix \
        sockets

# Composer
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /var/www/html

# Production-ish PHP/OPcache tuning for Octane.
RUN { \
        echo "opcache.enable=1"; \
        echo "opcache.enable_cli=1"; \
        echo "opcache.jit=1255"; \
        echo "opcache.jit_buffer_size=64M"; \
        echo "opcache.memory_consumption=192"; \
        echo "opcache.max_accelerated_files=20000"; \
        echo "opcache.validate_timestamps=1"; \
        echo "memory_limit=512M"; \
    } > "$PHP_INI_DIR/conf.d/zz-octane.ini"

# Copy the application source
COPY src/ /var/www/html/

# Install PHP dependencies
RUN composer install --no-interaction --prefer-dist --optimize-autoloader || true

# Permissions for Laravel writable dirs
RUN chown -R www-data:www-data /var/www/html \
    && chmod -R 775 storage bootstrap/cache

COPY docker/php/entrypoint.sh /usr/local/bin/entrypoint
RUN chmod +x /usr/local/bin/entrypoint

EXPOSE 8000

ENTRYPOINT ["entrypoint"]

# Boot the app once and serve it via FrankenPHP workers (Octane).
CMD ["php", "artisan", "octane:start", \
     "--server=frankenphp", \
     "--host=0.0.0.0", \
     "--port=8000", \
     "--workers=4", \
     "--max-requests=500"]
