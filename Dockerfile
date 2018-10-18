FROM golang:1.11-alpine AS build

WORKDIR /go/src/app

RUN apk add --update git && go get github.com/golang/dep/cmd/dep

COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

COPY . .

RUN go build -v

FROM alpine
WORKDIR /app
COPY --from=build /go/src/app/app /app

CMD [ "./app" ]