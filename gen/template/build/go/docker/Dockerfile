FROM alpine:latest
LABEL maintainer="adrianwit<adrianwit@gmail.com>"
RUN apk add --no-cache ca-certificates bash
RUN apk add --update tzdata curl && rm -rf /var/cache/apk/*
ADD $app /
$assets
CMD ["/$app"$args]