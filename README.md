# ghbot

Github bot, primarily for the collectd project.

## About

**ghbot** is a bot to automate tasks on Github, primarily to react to issues and
pull requests.

## Setup

1.  Create a *Personal access token* for the Github user you want the bot to act
    as. To do so, go to *Your profile → Settings →  Developer settings →  Personal access
    tokens*. This is the *access token* for step 3.
2.  Create a webhook under *Your repository → Settings → Webhooks*. Provide a
    shared secret with which the webhook will sign its request. For example:

        dd if=/dev/random bs=64 count=1 status=none | sha256sum

    This is the *secret key* for step 3.
3.  Create a *Datastore Entity* with the *access token* and *secret key*:
    *   Kind: `credentials`
    *   Key identifier: `singleton` (string)
    *   Properties
        *   `AccessToken`: *access token* (string)
        *   `SecretKey`: *secret key* (string)
4.  Deploy to App Engine:

        gcloud app deploy --version="v$(date +%s)"

## License

[ISC License](https://opensource.org/licenses/ISC)

## Author

Florian Forster
