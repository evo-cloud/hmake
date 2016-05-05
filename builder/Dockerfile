FROM golang:1.6-alpine
RUN apk update && apk add git && rm -fr /var/cache/apk/* && \
    go get -v github.com/FiloSottile/gvt && \
    go get -v github.com/onsi/ginkgo/ginkgo && \
    go get -v github.com/onsi/gomega
