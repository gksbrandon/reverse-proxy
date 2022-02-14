FROM golang:1.12.0-alpine3.9 AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o main .

FROM alpine
WORKDIR /app
COPY --from=builder /app/ /app/
CMD ["/app/main"]
