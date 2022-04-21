FROM golang:1.18.1-alpine3.15 AS builder
WORKDIR /app
COPY . ./

RUN apk add --no-cache git

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./build/cms ./cmd/cms
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./build/send ./cmd/send
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./build/web ./cmd/web

FROM alpine:3.15 AS cms
WORKDIR /app
COPY --from=0 /app/build/cms ./
CMD ["./cms"]

FROM alpine:3.15 AS send
WORKDIR /app
COPY --from=0 /app/build/send ./
CMD ["./send"]

FROM alpine:3.15 AS web
WORKDIR /app
COPY --from=0 /app/build/web ./
COPY --from=0 /app/html ./html
COPY --from=0 /app/public ./public
CMD ["./web"]
