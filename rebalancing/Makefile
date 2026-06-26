.PHONY: setup start stop migrate test

setup:
	docker-compose up -d postgres redpanda
	sleep 5
	psql -U postgres -h localhost -f schema.sql
	docker-compose up -d temporal hasura

start:
	docker-compose up -d
	go run worker/main.go &
	go run api/main.go &

stop:
	docker-compose down
	pkill -f "go run"

migrate:
	psql -U postgres -h localhost -d portfolio -f migrations/*.sql

test:
	go test ./... -v

clean:
	docker-compose down -v
	rm -rf postgres_data

logs:
	docker-compose logs -f

redpanda-setup:
	# Create topics in Redpanda (requires rpk inside the container)
	docker-compose exec redpanda rpk topic create orders --partitions 1 || true
	docker-compose exec redpanda rpk topic create notifications --partitions 1 || true
	docker-compose exec redpanda rpk topic create order.execution --partitions 1 || true