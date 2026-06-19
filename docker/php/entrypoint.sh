#!/bin/sh
set -e

cd /var/www/html

# Ensure dependencies exist (in case of mounted volume on first run)
if [ ! -d vendor ]; then
    composer install --no-interaction --prefer-dist --optimize-autoloader
fi

# Create .env from example if missing
if [ ! -f .env ]; then
    cp .env.example .env
fi

# Generate app key if not set
if ! grep -q "^APP_KEY=base64" .env 2>/dev/null; then
    php artisan key:generate --force
fi

# Wait for Postgres to be ready
echo "Waiting for database at ${DB_HOST}:${DB_PORT}..."
until php -r "exit(@fsockopen(getenv('DB_HOST'), (int)getenv('DB_PORT')) ? 0 : 1);" 2>/dev/null; do
    sleep 1
done
echo "Database is up."

# Run migrations + seeders (idempotent enough for a fresh marketplace)
php artisan migrate --force
php artisan db:seed --force || true

# Storage symlink for public files
php artisan storage:link || true

# Ensure file-based cache/session directories exist and are writable
mkdir -p storage/framework/cache/data storage/framework/sessions storage/framework/views
chown -R www-data:www-data storage bootstrap/cache || true

exec "$@"
