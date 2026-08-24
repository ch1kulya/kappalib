FROM golang:1.27.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/a-h/templ/cmd/templ@$(go list -m -f '{{.Version}}' github.com/a-h/templ)

COPY . .

RUN templ generate

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o server ./cmd/server

FROM alpine:3.23

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/server .
COPY --from=builder --chown=appuser:appgroup /app/assets ./assets
COPY --from=builder --chown=appuser:appgroup /app/migrations ./migrations

USER appuser

EXPOSE 1666

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:1666/healthz || exit 1

CMD ["./server"]
