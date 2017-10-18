#!/bin/bash

set -e

declare -r ACCT='octo@verplant.org'
declare -r PROJ='collectd-github-bot'

declare VERSION="v$(date +%s)"
declare TAG="gcr.io/collectd-github-bot/github-bot:${VERSION}"

docker build -t "${TAG}" .
gcloud docker --account="${ACCT}" --project="${PROJ}" -- push "${TAG}"

gcloud app deploy --account="${ACCT}" --project="${PROJ}" --version="${VERSION}" --image-url "${TAG}"
