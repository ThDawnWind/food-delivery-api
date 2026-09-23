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
