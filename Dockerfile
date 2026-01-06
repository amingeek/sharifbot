# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o telegram-bot ./main.go

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install required packages
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/sharifbot .

# Copy configuration and data directories
COPY --from=builder /app/data ./data
COPY --from=builder /app/.env.example ./.env

# Create necessary directories
RUN mkdir -p ./data/uploads ./logs

# Set timezone to Tehran
ENV TZ=Asia/Tehran

# Expose ports
EXPOSE 8080 8081 8082

# Run the application
CMD ["./sharifbot"]