#!/bin/sh
set -e

echo "🚀 WhatsApp Meta API Adapter - Starting..."

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL..."
until pg_isready -h postgres -U whatsapp -d whatsapp_adapter > /dev/null 2>&1; do
  echo "   Postgres is unavailable - sleeping"
  sleep 2
done
echo "✅ PostgreSQL is ready"

# Install golang-migrate if not present (for migration execution)
if ! command -v migrate > /dev/null 2>&1; then
  echo "📦 Installing golang-migrate..."
  apk add --no-cache curl
  curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
  mv migrate /usr/local/bin/migrate
  chmod +x /usr/local/bin/migrate
fi

# Run database migrations
echo "🔄 Running database migrations..."
if migrate -path /app/migrations -database "$DATABASE_URL" up; then
  echo "✅ Migrations completed successfully"
else
  echo "⚠️  Migration failed or already up to date"
  # Don't fail - migrations might already be applied
fi

# Start the application
echo "🎯 Starting application server..."
exec /app/server
