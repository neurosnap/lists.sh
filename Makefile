BIN=./node_modules/.bin
PGDATABASE?="lists"
PGHOST?="db"
PGUSER?="postgres"
PORT?="5432"
DB_CONTAINER?=listssh_db_1

build:
	go build -o build/web ./cmd/web
	go build -o build/cms ./cmd/cms
	go build -o build/send ./cmd/send
.PHONY: build

create:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) < ./db/setup.sql
.PHONY: create

teardown:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/teardown.sql
.PHONY: teardown

migrate:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220310_init.sql
.PHONY: migrate

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

image-build:
	docker build -t neurosnap/lists-cms --target cms .
	docker build -t neurosnap/lists-send --target send .
	docker build -t neurosnap/lists-web --target web .
.PHONY: build

image-push:
	docker push neurosnap/lists-cms
	docker push neurosnap/lists-send
	docker push neurosnap/lists-web
.PHONY: push

bp: image-build image-push
.PHONY: bp

deploy:
	docker system prune -f
	docker-compose -f production.yml pull --ignore-pull-failures
	docker-compose -f production.yml up --no-deps -d
.PHONY: deploy
