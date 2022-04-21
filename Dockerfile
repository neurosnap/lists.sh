FROM golang:1.18.1-alpine3.15 AS builder
WORKDIR /app
COPY . ./

RUN apk add --no-cache git

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./build/ssh ./cmd/ssh
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./build/web ./cmd/web

FROM alpine:3.15 AS ssh
WORKDIR /app
COPY --from=0 /app/build/ssh ./
CMD ["./ssh"]

FROM alpine:3.15 AS web
WORKDIR /app
COPY --from=0 /app/build/web ./
COPY --from=0 /app/html ./html
COPY --from=0 /app/public ./public
CMD ["./web"]
