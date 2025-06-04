# Start from the official Golang image for building
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the proto files
COPY proto/ ./proto/

# Copy the TLS Certificates files
COPY certs/ ./certs/

# Copy the environment file
COPY server/.env ./

# Copy the server source code
COPY server/main.go ./server/

# Build the Go binary
RUN go build -o grpc-server ./server/main.go

# Expose the port your server listens on (change if needed)
EXPOSE 50051

# If both the .env file and this variable are present, the value from the variable will override the value from the .env file.
ENV PORT=":50051"
ENV TLS_ENABLED="false"
# Certificate and key files needed when tls is enabled
ENV TLS_CERT_FILE=""
ENV TLS_KEY_FILE=""
ENV LOG_FILE=""

# Run the binary
CMD ["./grpc-server"]

# ENTRYPOINT ["/app/myapp"]

# docker run -e VAR1="Value1" -e VAR2="Value2" -e VAR3="Value3" my-golang-app
