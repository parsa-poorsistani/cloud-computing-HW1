# Start from the latest Golang base image
FROM golang:alpine AS builder

# Set the current working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source from the current directory to the working directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .
# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["/app/main"]
