# Stage 1: Build the Go application
FROM golang:1.17 AS builder

# Setting the working directory to /app
WORKDIR /app

# Copy local files to working directory in container
COPY go.mod go.sum ./
RUN go get github.com/google/go-cmp/cmp && go mod download
COPY . .

# Build the Go application, producing an executable named "admissioncontroller"
RUN go build -o admissioncontroller main.go

# Stage 2: Create the runtime container
FROM alpine:latest

# Set the working directory in the runtime container
WORKDIR /root/

# Copy the executable from the builder stage into the runtime container
COPY --from=builder /app/admissioncontroller .

# Expose port 8080 for the application
EXPOSE 8080

# Command to run the application
CMD ["./admissioncontroller"]

