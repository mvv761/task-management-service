FROM golang:1.26.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o api ./cmd/api

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/api ./api

CMD ["./api"]