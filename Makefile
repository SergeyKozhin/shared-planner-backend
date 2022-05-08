DB_DSN=postgres://postgres:plannerpassword@localhost/postgres?sslmode=disable

migrate_up:
	migrate -path=./migrations -database=${DB_DSN} up ${VERSION}

migrate_down:
	migrate -path=./migrations -database=${DB_DSN} down ${VERSION}

migrate_force:
	migrate -path=./migrations -database=${DB_DSN} force ${VERSION}

migrate_create:
	migrate create -seq -ext=.sql -dir=./migrations ${NAME}

docker_tag:
	docker tag shared-planner-backend_app ghcr.io/sergeykozhin/shared-planner-backend:latest

docker_push:
	docker push ghcr.io/sergeykozhin/shared-planner-backend:latest