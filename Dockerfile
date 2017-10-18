FROM alpine:3.6
MAINTAINER  Florian Forster <ff@octo.it>

RUN apk add --no-cache bash clang
COPY ./bot /opt/github-bot/bin/github-bot

ENTRYPOINT ["/opt/github-bot/bin/github-bot"]
EXPOSE 8080
