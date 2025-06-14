# Debug Dockerfile
FROM golang:1.22

WORKDIR /app

# Copy everything
COPY . .

# Initialize go module if needed
RUN go mod init bookstore-service || true

# Download dependencies
RUN go mod tidy

# Try to build
RUN go build -o main .

EXPOSE 8080
CMD ["./main"]