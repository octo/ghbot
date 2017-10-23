# ghbot

Github bot, primarily for the collectd project.

## About

**ghbot** is a bot to automate tasks on Github, primarily to react to issues and
pull requests.

## Setup

1.  Create a *Personal access token* for the Github user you want the bot to act
    as. To do so, go to *Your profile → Settings →  Developer settings →  Personal access
    tokens*. Copy this token to the `accessToken` variable in
    `client/client.go`.
2.  Create a webhook under *Your repository → Settings → Webhooks*. Provide a
    shared secret with which the webhook will sign its request. For example:

        dd if=/dev/random bs=64 count=1 status=none | sha256sum

    Copy this secret to the `secretKey` variable in `ghbot.go`.
3.  Deploy to App Engine:

        gcloud app deploy --version="v$(date +%s)"

## License

[ISC License](https://opensource.org/licenses/ISC)

## Author

Florian Forster
