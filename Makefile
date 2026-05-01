.PHONY: prod dev logs codegen clean

prod:
	docker compose up --build

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
	docker compose -f docker-compose.yml -f docker-compose.dev.yml watch &
	docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

logs:
	docker compose logs -f $(service)

codegen:
	docker compose run --rm codegen

clean:
	docker compose down --rmi local --volumes --remove-orphans
