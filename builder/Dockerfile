FROM golang:1.6-alpine
RUN apk update && apk add git tar docker && rm -fr /var/cache/apk/* && \
    go get -v github.com/FiloSottile/gvt && \
    go get -v github.com/onsi/ginkgo/ginkgo && \
    go get -v github.com/onsi/gomega && \
    chmod -R a+rw /go
