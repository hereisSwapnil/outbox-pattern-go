# Justfile
default: help

# Show help
help:
	@just --list

# Start the full stack
start:
	docker-compose up -d --build
	@echo "Services started! App is at http://localhost:8080"
	@echo "Run 'just logs' to see the relay output."

# Stop the stack
stop:
	docker-compose down -v

# View logs for the relay
logs:
	docker-compose logs -f relay

# Test order insertion
test-order item="Mechanical Keyboard":
	curl -X POST http://localhost:8080/orders \
		-H "Content-Type: application/json" \
		-d '{"item": "{{item}}"}'
