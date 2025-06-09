FROM golang:1.24.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o blackjack

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/blackjack .

CMD ["./blackjack"] 