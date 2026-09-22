FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/migrate ./cmd/migrate


FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/api ./api
COPY --from=builder /app/migrate ./migrate
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./api"]
