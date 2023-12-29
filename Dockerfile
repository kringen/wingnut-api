FROM golang:1.21 AS build-stage

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /wingnut-api

# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application binary into a lean image
#FROM gcr.io/distroless/base-debian11 AS build-release-stage
FROM registry.access.redhat.com/ubi9/ubi-minimal
WORKDIR /

COPY --from=build-stage /wingnut-api /wingnut-api

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
#EXPOSE 8080

#USER nonroot:nonroot
USER 1001

ENTRYPOINT ["/wingnut-api"]