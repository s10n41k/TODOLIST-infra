.PHONY: help start stop restart logs clean init

help:
	@echo "TODOLIST Infrastructure Management"
	@echo ""
	@echo "Commands:"
	@echo "  make init     - Initialize environment"
	@echo "  make start    - Start all services"
	@echo "  make stop     - Stop all services"
	@echo "  make restart  - Restart all services"
	@echo "  make logs     - Show logs"
	@echo "  make clean    - Clean up (remove containers and volumes)"
	@echo "  make health   - Check services health"

init:
	@echo "🚀 Initializing infrastructure..."
	@if [ ! -f .env ]; then \
		echo "📝 Creating .env file from template..."; \
		cp .env.example .env; \
		echo "✅ Please edit .env file with your secrets"; \
	else \
		echo "✅ .env file already exists"; \
	fi

start:
	@echo "🚀 Starting all services..."
	@docker-compose up -d
	@echo "✅ Services started"
	@echo "🌐 Gateway: http://localhost:8080"

stop:
	@echo "🛑 Stopping all services..."
	@docker-compose down
	@echo "✅ Services stopped"

restart: stop start

logs:
	@docker-compose logs -f

clean:
	@echo "🧹 Cleaning up..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@echo "✅ Cleanup completed"

health:
	@echo "🏥 Health check..."
	@./scripts/health-check.sh

status:
	@echo "📊 Services status:"
	@docker-compose ps

build:
	@echo "🔨 Building all services..."
	@docker-compose build --no-cache
	@echo "✅ Build completed"