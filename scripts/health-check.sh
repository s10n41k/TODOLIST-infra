#!/bin/bash

echo "🔍 Health check for TODOLIST services..."

SERVICES=(
    "Gateway:8080"
    "Auth:8787"
    "Users:8081"
    "Tasks:8082"
    "Analytics:8083"
)

for service in "${SERVICES[@]}"; do
    name=$(echo $service | cut -d':' -f1)
    port=$(echo $service | cut -d':' -f2)

    if curl -s -f http://localhost:$port/health > /dev/null 2>&1; then
        echo "✅ $name (port $port) is healthy"
    elif curl -s -f http://localhost:$port/ > /dev/null 2>&1; then
        echo "⚠️  $name (port $port) is running but /health not found"
    else
        echo "❌ $name (port $port) is NOT responding"
    fi
done

echo ""
echo "📊 Database status:"
if docker-compose ps | grep -q "Up"; then
    echo "✅ Docker containers are running"
else
    echo "❌ Some containers are not running"
fi