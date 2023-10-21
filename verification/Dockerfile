# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.17

# Set the working directory in the container
WORKDIR /go/src/my-go-app

# Copy the local package files to the container's workspace.
COPY . /go/src/my-go-app

# Fetch dependencies.
# If the Go Modules are enabled, this will also download the dependencies
# specified in the go.mod and go.sum files.
RUN go mod download

# Build the my-go-app command inside the container.
# This will build the binaries of your Go app.
RUN go build -o my-go-app .

# Run the my-go-app command by default when the container starts.
ENTRYPOINT ["/go/src/my-go-app/my-go-app"]

# Document that the service listens on port 8080.
EXPOSE 8080
