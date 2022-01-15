# Build the manager binary
FROM golang:1.16 as build

WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod .
COPY go.sum .

# cache deps
RUN go mod download

COPY main.go .
COPY controller.go .
COPY pkg ./pkg

# Build, CGO must be enabled since since confluentinc/confluent-kafka-go is backed by C client
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o /kafkaops-controller .

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /kafkaops-controller /kafkaops-controller

USER nonroot:nonroot

ENTRYPOINT ["/kafkaops-controller"]