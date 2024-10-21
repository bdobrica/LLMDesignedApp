# Step 1: Use the official Go image for building the app
FROM golang:1.23.2-alpine AS builder

# Set working directory inside the container
WORKDIR /app

# Copy the Go modules manifest and download dependencies first (to benefit from layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the Go binary
RUN go build -o user-management

# Step 2: Use a smaller base image for running the app
FROM alpine:latest

# Copy the built binary from the builder stage
COPY --from=builder /app/user-management /app/user-management

# Set working directory
WORKDIR /app

# Expose port 3000 for user-management service
EXPOSE 3000

# Start the service
CMD ["./user-management"]
