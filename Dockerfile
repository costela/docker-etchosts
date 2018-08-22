FROM golang:1.10-alpine

WORKDIR /go/src/app

RUN apk add --update git && go get github.com/golang/dep/cmd/dep

COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

COPY . .

RUN go install -v

CMD ["app"]