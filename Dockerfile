## Build
FROM golang:1.17.7-alpine3.15 AS builder

WORKDIR /app

COPY src/go.mod ./
COPY src/go.sum ./
RUN go mod download

COPY src/*.go ./
RUN go build -o /reverse-proxy

## Deploy
FROM alpine
WORKDIR /
COPY --from=builder /reverse-proxy /reverse-proxy

EXPOSE 8080

CMD ["/reverse-proxy"]
