FROM golang:1.7-alpine
RUN apk update && apk add bash curl git tar zip && rm -fr /var/cache/apk/* && \
    curl -sSL https://get.docker.com/builds/Linux/x86_64/docker-1.10.0.tgz | tar -C / -xz && \
    curl -sSL https://github.com/spf13/hugo/releases/download/v0.16/hugo_0.16_linux-64bit.tgz | tar -C /usr/local/bin -xz && \
    rm -f /usr/local/bin/README.md /usr/local/bin/LICENSE.md && \
    go get -v github.com/alecthomas/gometalinter && \
    go get -v golang.org/x/tools/cmd/... && \
    go get -v github.com/FiloSottile/gvt && \
    go get -v github.com/onsi/ginkgo/ginkgo && \
    go get -v github.com/onsi/gomega && \
    gometalinter --install && \
    chmod -R a+rw /go
