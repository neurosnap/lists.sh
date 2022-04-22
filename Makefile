BIN=./node_modules/.bin
PGDATABASE?="lists"
PGHOST?="db"
PGUSER?="postgres"
PORT?="5432"
DB_CONTAINER?=listssh_db_1

build:
	go build -o build/web ./cmd/web
	go build -o build/ssh ./cmd/ssh
.PHONY: build

create:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) < ./db/setup.sql
.PHONY: create

teardown:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/teardown.sql
.PHONY: teardown

migrate:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220310_init.sql
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220422_add_desc_to_user_and_post.sql
.PHONY: migrate

latest:
	docker exec -i $(DB_CONTAINER) psql -U $(PGUSER) -d $(PGDATABASE) < ./db/migrations/20220422_add_desc_to_user_and_post.sql
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

image-build:
	docker build -t neurosnap/lists-ssh --target ssh .
	docker build -t neurosnap/lists-web --target web .
.PHONY: build

image-push:
	docker push neurosnap/lists-ssh
	docker push neurosnap/lists-web
.PHONY: push

bp: image-build image-push
.PHONY: bp

deploy:
	docker system prune -f
	docker-compose -f production.yml pull --ignore-pull-failures
	docker-compose -f production.yml up --no-deps -d
.PHONY: deploy
