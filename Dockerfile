# syntax=docker/dockerfile:1
# Based on https://docs.docker.com/language/golang/build-images/#multi-stage-builds

# Build the application from source
FROM golang:1.24

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

ARG CGO_ENABLED=0
ARG GOARCH=amd64
ARG GOOS=linux
RUN go build -ldflags '-s -w' -trimpath -o /docker-etchosts

# Deploy the application binary into a lean image
FROM scratch AS build-release-stage

WORKDIR /

COPY --from=0 /docker-etchosts /docker-etchosts

ENTRYPOINT ["/docker-etchosts"]
