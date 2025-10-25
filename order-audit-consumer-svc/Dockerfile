FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY order-audit-consumer-svc/go.mod order-audit-consumer-svc/go.sum ./order-audit-consumer-svc/
COPY order-svc/go.mod order-svc/go.sum ./order-svc/

WORKDIR /app/order-audit-consumer-svc

RUN go mod download

COPY order-audit-consumer-svc/ .
COPY order-svc/ ../order-svc/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./main ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/order-audit-consumer-svc/main ./main

CMD ["./main"]
