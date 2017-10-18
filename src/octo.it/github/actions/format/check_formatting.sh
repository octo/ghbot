#!/bin/bash

set -e
#set -x

if [[ $# != 4 ]]; then
  echo "Usage: $0 <owner> <repo> <branch> <base>"
  exit 1
fi

declare -r OWNER="$1"; shift
declare -r REPO="$1"; shift
declare -r BRANCH="$1"; shift
declare -r BASE="$1"; shift

if [[ ! -d /var/lib/check_formatting ]]; then
  mkdir /var/lib/check_formatting
  cd /var/lib/check_formatting
  git clone -q --origin github git://github.com/collectd/collectd.git
fi

cd /var/lib/check_formatting/collectd

git remote update github >/dev/null
git fetch -f -q "git://github.com/${OWNER}/${REPO}" "${BRANCH}:check_formatting"
git checkout -q check_formatting

#declare -r FILE_ORIG="$(tempfile)"
#declare -r FILE_FRMT="$(tempfile)"

git diff --name-only "github/${BASE}...FETCH_HEAD" | egrep '\.(c|h|proto)$' | while read f; do
  clang-format -style=file -i "${f}"
done

declare -i HAVE_CHANGES=0
if ! git diff --quiet; then
  git diff | cat
  HAVE_CHANGES=1
fi

git reset -q --hard
git checkout -q master
git branch -q -D check_formatting

exit $HAVE_CHANGES
