PGDATABASE?="lists"
PGHOST?="db"
PGUSER?="postgres"
PORT?="5432"
DB_CONTAINER?=listssh_db_1
DOCKER_TAG?=$(shell git log --format="%H" -n 1)

test:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -E goimports -E godot
.PHONY: test

build:
	go build -o build/web ./cmd/web
	go build -o build/ssh ./cmd/ssh
	go build -o build/gemini ./cmd/gemini
.PHONY: build

format:
	go fmt ./...
.PHONY: format

create:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) < ./db/setup.sql
.PHONY: create

teardown:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/teardown.sql
.PHONY: teardown

migrate:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220310_init.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220422_add_desc_to_user_and_post.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220426_add_index_for_filename.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220427_username_to_lower.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220523_timestamp_with_tz.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220721_analytics.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220722_post_hidden.sql
.PHONY: migrate

latest:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220721_analytics.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220722_post_hidden.sql
.PHONY: latest

psql:
	docker exec -it $(DB_CONTAINER) psql -U $(PGUSER)
.PHONY: psql

dump:
	docker exec -it $(DB_CONTAINER) pg_dump -U $(PGUSER) $(PGDATABASE) > ./backup.sql
.PHONY: dump

restore:
	docker cp ./backup.sql $(DB_CONTAINER):/backup.sql
	docker exec -it $(DB_CONTAINER) /bin/bash
	# psql postgres -U postgres < /backup.sql
.PHONY: restore

bp-setup:
	docker buildx ls | grep pico || docker buildx create --name pico
	docker buildx use pico
.PHONY: bp-setup

bp-caddy: bp-setup
	docker buildx build --push --platform linux/amd64,linux/arm64 -t neurosnap/lists-caddy:$(DOCKER_TAG) -f Dockerfile.caddy .
.PHONY: bp-caddy

bp-ssh: bp-setup
	docker buildx build --push --platform linux/amd64,linux/arm64 -t neurosnap/lists-ssh:$(DOCKER_TAG) --target ssh .
.PHONY: bp-ssh

bp-web: bp-setup
	docker buildx build --push --platform linux/amd64,linux/arm64 -t neurosnap/lists-web:$(DOCKER_TAG) --target web .
.PHONY: bp-web

bp-gemini: bp-setup
	docker buildx build --push --platform linux/amd64,linux/arm64 -t neurosnap/lists-gemini:$(DOCKER_TAG) --target gemini .
.PHONY: bp-gemini

bp: bp-ssh bp-web bp-caddy
.PHONY: bp

deploy:
	docker system prune -f
	docker-compose -f production.yml pull --ignore-pull-failures
	docker-compose -f production.yml up --no-deps -d
.PHONY: deploy
