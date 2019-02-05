#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(dirname "${BASH_SOURCE}")/..

GO_VERSION=($(go version))

if [[ -z $(echo "${GO_VERSION[2]}" | grep -E 'go1.9|go1.10|go1.11') ]]; then
  echo "Unknown go version '${GO_VERSION}', skipping gofmt."
  exit 0
fi

cd "${ROOT}"

find_files() {
  find . -not \( \
      \( \
        -wholename './output' \
        -o -wholename './_output' \
        -o -wholename './release' \
        -o -wholename './target' \
        -o -wholename '*/third_party/*' \
        -o -wholename '*/Godeps/*' \
        -o -wholename '*/vendor/*' \
      \) -prune \
    \) -name '*.go'
}

GOFMT="gofmt -s -w"
bad_files=$(find_files | xargs $GOFMT -l)
if [[ -n "${bad_files}" ]]; then
  echo "!!! '$GOFMT' needs to be run on the following files: "
  echo "${bad_files}"
  exit 1
fi

GOFMT="gofmt -s"
bad_files=$(find_files | xargs $GOFMT -l)
if [[ -n "${bad_files}" ]]; then
  echo "!!! '$GOFMT' needs to be run on the following files: "
  echo "${bad_files}"
  exit 1
fi
