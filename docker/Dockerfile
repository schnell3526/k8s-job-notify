# syntax=docker/dockerfile:1

# Dev stage
FROM mcr.microsoft.com/devcontainers/go:1.25 AS dev
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o /app/k8s-job-notify \
    ./cmd

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /app/k8s-job-notify /app/k8s-job-notify
USER nonroot:nonroot
ENTRYPOINT ["/app/k8s-job-notify"]
