default:
    @just --list

prod:
    docker compose up --build -d

dev:
    docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
    docker compose -f docker-compose.yml -f docker-compose.dev.yml watch &
    docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

logs service:
    docker compose logs -f {{ service }}

clean:
    docker compose down --rmi local --volumes --remove-orphans

codegen:
    docker compose --profile codegen run --rm codegen

server-test:
    docker compose --profile test run --rm --build server-test

prune:
    docker system prune -a --volumes

genkeys:
    cd server && go run ./cmd/genkeys

signnonce nonce:
    cd server && go run ./cmd/signnonce {{ nonce }}
