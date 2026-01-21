#!/bin/bash

echo "🚀 Starting TODOLIST microservices..."

# Проверяем .env файл
if [ ! -f ../.env ]; then
    echo "❌ .env file not found!"
    echo "Please copy .env.example to .env and fill in the values"
    exit 1
fi

# Запускаем
docker-compose up -d

echo "⏳ Waiting for services to start..."
sleep 5

echo ""
echo "✅ Services started successfully!"
echo ""
echo "🌐 Gateway:      http://localhost:8080"
echo "🔐 Auth:         http://localhost:8787"
echo "👥 Users:        http://localhost:8081"
echo "📝 Tasks:        http://localhost:8082"
echo "📊 Analytics:    http://localhost:8083"
echo ""
echo "📈 Check status: make health"
echo "📋 View logs:    make logs"