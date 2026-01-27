# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o switch-gate ./cmd/switch-gate

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/switch-gate /usr/local/bin/

# Create config directory
RUN mkdir -p /etc/switch-gate

EXPOSE 18388 9090

ENTRYPOINT ["switch-gate"]
CMD ["-config", "/etc/switch-gate/config.yaml"]
