# Start from the latest Golang base image
FROM golang:latest AS builder

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
RUN ls
RUN ls -al /app
# Use a minimal alpine image for the final stage of the build
FROM alpine:latest

# Copy the binary from the builder stage to the current stage
COPY --from=builder /app /app

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["/app/main"]
