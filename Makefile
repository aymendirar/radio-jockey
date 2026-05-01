.PHONY: prod dev codegen clean

prod:
	docker compose up --build

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml watch

codegen:
	docker compose run --rm codegen

clean:
	docker compose down --rmi local --volumes --remove-orphans
