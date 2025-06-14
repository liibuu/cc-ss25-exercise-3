# Simple Dockerfile - use this for all services
FROM golang:1.22-alpine

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]