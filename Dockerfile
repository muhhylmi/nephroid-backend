# 1. Build Stage
FROM golang:alpine AS builder

# Install required system packages
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files to download dependencies first 
# (this leverages Docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 ensures a static binary is created, perfect for Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o nephroid-api ./cmd/api

# 2. Final Minimal Image Stage
FROM alpine:latest

# Install CA certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/nephroid-api .

# Expose the port the API runs on
EXPOSE 8080

# Command to run the application
CMD ["./nephroid-api"]
