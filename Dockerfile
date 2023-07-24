# Start from the full Golang image to build the binary
FROM golang:1.20-alpine3.18 as builder

# Define ARG for source commit with default value as N/A
ARG SOURCE_COMMIT=N/A

# Set the current working directory inside the container
WORKDIR /app

# Download the necessary Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the Go source code into the container
COPY . .

# Build the Go application with SOURCE_COMMIT as build argument
RUN DATE=$(date -u '+%Y-%m-%d-%H%M UTC'); \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o aws_health_exporter -ldflags="-X 'main.Version=${SOURCE_COMMIT}' -X 'main.BuildTime=${DATE}'" .

# Use the Google Distroless image for the final image
FROM gcr.io/distroless/base

# Copy the binary from builder
COPY --from=builder /app/aws_health_exporter .

# Run the binary
ENTRYPOINT ["./aws_health_exporter"]
