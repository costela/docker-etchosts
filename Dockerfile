FROM golang:1.13-alpine AS build

RUN apk add --update git

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go /app/

RUN go build -v

FROM alpine

COPY --from=build /app/docker-etchosts /docker-etchosts

CMD [ "/docker-etchosts" ]