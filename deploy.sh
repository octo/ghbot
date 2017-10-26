#!/bin/bash

set -e

declare -r ACCT='octo@verplant.org'
declare -r PROJ='collectd-github-bot'

declare -r VERSION="v$(date +%s)"

export CLOUDSDK_CORE_DISABLE_PROMPTS=1

gcloud app deploy --account="${ACCT}" --project="${PROJ}" --version="${VERSION}"
