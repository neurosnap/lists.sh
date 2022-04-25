# lists.sh

A microblog for lists.

## Setup

- golang `v1.18`

```bash
make build
```

## Run

There are two apps: an ssh and web server.

```bash
./build/ssh
```

Default port for ssh server is `2222`.

```bash
./build/web
```

Default port for web server is `3000`.

## Deployment

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
