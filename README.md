# lists.sh

A microblog for lists.

## comms

- [website](https://pico.sh)
- [irc #pico.sh](irc://irc.libera.chat/#pico.sh)
- [mailing list](https://lists.sr.ht/~erock/pico.sh)
- [ticket tracker](https://todo.sr.ht/~erock/pico.sh)
- [email](mailto:hello@pico.sh)

## setup

- golang `v1.18`

You'll also need some environment variables

```
export POSTGRES_PASSWORD="secret"
export DATABASE_URL="postgresql://postgres:secret@db/lists?sslmode=disable"
export LISTS_SSH_PORT=2222
export LISTS_WEB_PORT=3000
export LISTS_DOMAIN="lists.sh"
export LISTS_EMAIL="support@lists.sh"
export LISTS_PROTOCOL="http"
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

### subdomains

Since we use subdomains for blogs, you'll need to update your `/etc/hosts` file
to accommodate.

```bash
# /etc/hosts
127.0.0.1 lists.test
127.0.0.1 erock.lists.test
```

Wildcards are not support in `/etc/hosts` so you'll have to add a subdomain for
each blog in development. For this example you'll also want to change the domain 
env var to `LISTS_DOMAIN=lists.test`.

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
