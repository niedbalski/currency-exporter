FROM golang:alpine as build
MAINTAINER  Jorge Niedbalski <jnr@metaklass.org>

RUN apk --no-cache add git

WORKDIR /go/src/github.com/niedbalski/currency-exporter
COPY . .
RUN go get github.com/tools/godep && godep restore && go build -o currency-exporter

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /go/src/github.com/niedbalski/currency-exporter/currency-exporter .

ENTRYPOINT ["./currency-exporter"]
