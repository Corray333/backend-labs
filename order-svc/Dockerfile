FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./cmd/main ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/cmd/main ./cmd/main
COPY --from=builder /app/migrations ./migrations


CMD ["./cmd/main"]