# Use the official Golang image as a base image
FROM golang:1.20 AS builder

# Set the working directory to the source code location
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Logstack Exporter binary
RUN go build -o logstack_exporter

# Use a minimal Alpine Linux image as the final image
FROM debian:bookworm-slim

# Set the working directory to /app
WORKDIR /app

# Copy the Logstack Exporter binary from the builder image
COPY --from=builder /app/logstack_exporter .

# Expose the default Prometheus metrics port
EXPOSE 9090

# Run Logstack Exporter when the container starts
ENTRYPOINT ["/app/logstack_exporter"]
