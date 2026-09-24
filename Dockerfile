FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/migrate ./cmd/migrate


FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=builder --chown=app:app /app/api ./api
COPY --from=builder --chown=app:app /app/migrate ./migrate
COPY --from=builder --chown=app:app /app/migrations ./migrations

USER app

EXPOSE 8080

CMD ["./api"]
