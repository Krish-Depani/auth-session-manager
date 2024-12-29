# Build stage
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev make

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache \
    postgresql15 \
    postgresql15-contrib \
    redis \
    ca-certificates \
    tzdata \
    su-exec \
    && mkdir -p /run/postgresql \
    && chown -R postgres:postgres /run/postgresql \
    && mkdir -p /var/lib/postgresql/data \
    && chown -R postgres:postgres /var/lib/postgresql/data

COPY --from=builder /app/main .
COPY --from=builder /app/.env .
COPY --from=builder /app/database/migrations ./database/migrations
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

COPY <<EOF /app/init-db.sql
CREATE USER root WITH PASSWORD 'root' SUPERUSER;
CREATE DATABASE asm;
GRANT ALL PRIVILEGES ON DATABASE asm TO root;
EOF

COPY <<EOF /app/init.sh
#!/bin/sh
set -e

# Initialize PostgreSQL
if [ ! -s /var/lib/postgresql/data/PG_VERSION ]; then
    echo "Initializing PostgreSQL database..."
    su-exec postgres initdb -D /var/lib/postgresql/data
    
    # Configure access
    echo "host all all 0.0.0.0/0 trust" >> /var/lib/postgresql/data/pg_hba.conf
    echo "listen_addresses='*'" >> /var/lib/postgresql/data/postgresql.conf
fi

# Start PostgreSQL
echo "Starting PostgreSQL..."
su-exec postgres postgres -D /var/lib/postgresql/data &

# Wait for PostgreSQL to start
echo "Waiting for PostgreSQL to start..."
until su-exec postgres pg_isready -h localhost -p 5432; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 1
done
echo "PostgreSQL is ready!"

# Initialize database and user
echo "Setting up database and user..."
su-exec postgres psql -f /app/init-db.sql

# Run migrations
echo "Running database migrations..."
export DATABASE_URL="postgres://root:root@localhost:5432/asm?sslmode=disable"
sleep 2  # Brief pause to ensure database is ready
migrate -path ./database/migrations -database "\${DATABASE_URL}" up

# Start Redis
echo "Starting Redis..."
redis-server --daemonize yes

# Wait for Redis
echo "Waiting for Redis to start..."
until redis-cli ping > /dev/null 2>&1; do
    echo "Redis is unavailable - sleeping"
    sleep 1
done
echo "Redis is ready!"

# Start the main application
echo "Starting main application..."
export GIN_MODE=release
./main
EOF

RUN chmod +x /app/init.sh

EXPOSE 3000

CMD ["/app/init.sh"]