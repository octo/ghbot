FROM alpine:3.6
MAINTAINER  Florian Forster <ff@octo.it>

RUN apk add --no-cache bash clang git
COPY src/octo.it/github/actions/format/check_formatting.sh /opt/github-bot/bin/
#COPY src/octo.it/github/bot/bot /opt/github-bot/bin/github-bot
COPY ./bot /opt/github-bot/bin/github-bot

ENTRYPOINT ["/opt/github-bot/bin/github-bot"]
EXPOSE 8080
