FROM golang:1.9-alpine
RUN apk update && apk add bash curl git tar zip py-pip nodejs nodejs-npm && rm -fr /var/cache/apk/* && \
    pip install 'docker-compose==1.8.0' && \
    npm install -g lunr-hugo && \
    curl -sSL https://get.docker.com/builds/Linux/x86_64/docker-1.10.0.tgz | tar -C / -xz && \
    curl -sSL https://github.com/spf13/hugo/releases/download/v0.20.1/hugo_0.20.1_Linux-64bit.tar.gz | tar -C /usr/local/bin -xz --strip-components=1 && \
    mv /usr/local/bin/hugo_0.20.1_linux_amd64 /usr/local/bin/hugo && \
    rm -f /usr/local/bin/README.md /usr/local/bin/LICENSE.md && \
    go get -v github.com/alecthomas/gometalinter && \
    go get -v golang.org/x/tools/cmd/... && \
    go get -v github.com/golang/dep/cmd/dep && \
    go get -v github.com/onsi/ginkgo/ginkgo && \
    go get -v github.com/onsi/gomega && \
    gometalinter --install && \
    chmod -R a+rw /go
