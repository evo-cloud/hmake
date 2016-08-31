FROM alpine:latest
RUN apk update && apk add docker && rm -fr /var/cache/apk/*
ADD amd64/hmake /usr/bin/
ENTRYPOINT ["/usr/bin/hmake"]
