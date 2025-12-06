# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies needed for build (e.g. git) if any
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# -ldflags="-w -s" reduces binary size by removing debug info
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/api/main.go

# Production Stage
FROM alpine:latest

WORKDIR /root/

# Install certificates for https calls and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/main .

# Copy .env file (optional: standard practice often passes env vars via orchestrator, but handy for simple docker runs)
# Ideally, we don't COPY .env in prod, but for convenience/completeness with user setup:
# COPY .env . 

# Expose port (adjust if needed, usually 8080 or from ENV)
EXPOSE 9090

# Command to run
CMD ["./main"]
