FROM golang:1.6-alpine
RUN apk update && apk add curl git tar zip && rm -fr /var/cache/apk/* && \
    curl -sSL https://get.docker.com/builds/Linux/x86_64/docker-1.10.0.tgz | tar -C / -xz && \
    go get -v github.com/alecthomas/gometalinter && \
    go get -v golang.org/x/tools/cmd/... && \
    go get -v github.com/FiloSottile/gvt && \
    go get -v github.com/onsi/ginkgo/ginkgo && \
    go get -v github.com/onsi/gomega && \
    go get -v github.com/spf13/hugo && \
    gometalinter --install && \
    chmod -R a+rw /go
