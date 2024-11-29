# Stage 1: Build the Go application
FROM golang:latest

# Setting the working directory to /app
WORKDIR /app

# Copy local files to working directory in container
COPY go.mod go.sum ./
RUN go mod tidy && go mod download
COPY . .

# Build the Go application, producing an executable named "admissioncontroller"
RUN go build -o admissioncontroller main.go

# Expose port 8080 for the application
EXPOSE 8080

# Command to run the application
CMD ["./admissioncontroller"]

