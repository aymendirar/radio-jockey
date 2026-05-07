.PHONY: prod dev logs codegen clean prune genkeys signnonce

prod:
	docker compose up --build

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
	docker compose -f docker-compose.yml -f docker-compose.dev.yml watch &
	docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

logs:
	docker compose logs -f $(service)

clean:
	docker compose down --rmi local --volumes --remove-orphans

codegen:
	docker compose run --rm codegen

prune:
	docker system prune -a --volumes

genkeys:
	cd server && go run ./cmd/genkeys

signnonce:
	cd server && go run ./cmd/signnonce $(nonce)
