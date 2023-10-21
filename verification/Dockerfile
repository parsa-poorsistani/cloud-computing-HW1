# Start from the latest golang base image
FROM golang:latest

# Set the current working directory inside the container
WORKDIR /usr/src/app

# Copy the local package files to the container’s workspace.
COPY . .
RUN go mod download
# Build the Go app
RUN go build -o main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Run the binary program produced by `go build`
CMD ["./main"]
