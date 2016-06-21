FROM golang:1.6-alpine
RUN apk add --update git make build-base && \
    git clone -b 3.3.6 https://github.com/sass/libsass /tmp/libsass && \
    git clone -b 3.3.6 https://github.com/sass/sassc /tmp/sassc && \
    cd /tmp/sassc && SASS_LIBSASS_PATH=/tmp/libsass make && cp bin/sassc /usr/local/bin/ && \
    rm -fr /tmp/libsass /tmp/sassc /var/cache/apk/*
