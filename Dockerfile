FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .

RUN templ generate

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

CMD ["./server"]
