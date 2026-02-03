# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copiar go.mod e go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copiar código
COPY . .

# Build binário
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o api ./cmd/api

# Runtime stage - minimal image
FROM alpine:3.19

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiar binário do builder
COPY --from=builder /app/api .

# Copiar migrations (opcional)
COPY migrations/ ./migrations/
COPY templates/ ./templates/

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Porta padrão
EXPOSE 8080

# User não-root por segurança
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Comando de inicialização
CMD ["./api"]
