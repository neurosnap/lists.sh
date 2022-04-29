# lists.sh

A microblog for lists.

## setup

- golang `v1.18`

You'll also need some environment variables

```
export POSTGRES_PASSWORD="secret"
export DATABASE_URL="postgresql://postgres:secret@db/lists?sslmode=disable"
export LISTS_SSH_PORT=2222
export LISTS_WEB_PORT=3000
```

I just use `direnv` which will load my `.env` file.

## development

### db

I use `docker-compose` to standup a postgresql server.  If you already have a
server running you can skip this step.

Copy example `.env`

```bash
cp .env.example .env
```

Then run docker compose.

```bash
docker-compose up -d
```

Then create the database and migrate

```bash
make create
make migrate
```

### build the apps

```bash
make build
```

### run the apps

There are two apps: an ssh and web server.

```bash
./build/ssh
```

Default port for ssh server is `2222`.

```bash
./build/web
```

Default port for web server is `3000`.

## deployment

I use `docker-compose` for deployment.  First you need `.env.prod`. 

```bash
cp .env.example .env.prod
```

The `production.yml` file in this repo uses my docker hub images for deployment.

```bash
docker-compose -f production.yml up -d
```

If you want to deploy using your own domain then you'll need to edit the
`Caddyfile` with your domain.
