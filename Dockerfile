FROM golang:1.19

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY cmd ./cmd
COPY pkg ./pkg

RUN go mod download
RUN go mod tidy

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o ./docker-tuple-space ./cmd/main.go

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/reference/dockerfile/#expose
EXPOSE 8080

# Run
CMD ["/docker-tuple-space"]