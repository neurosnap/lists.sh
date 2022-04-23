# lists.sh

A microblog for your lists.

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

I use `docker-compose` for deployment.  First you need `.env.prod`.  The
`production.yml` file in this repo uses my docker hub images for deployment.

```bash
cp .env.example .env.prod
```

```bash
docker-compose up -d
```
