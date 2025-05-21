#!/bin/bash

set -e

go install github.com/incu6us/goimports-reviser/v3@latest

PROJECT_NAME=github.com/apache/apisix-ingress-controller

find . -name '*.go' -print0 | while IFS= read -r -d '' file; do
  if [ "$file" != "./cmd/root/root.go" ]; then
    goimports-reviser -project-name "$PROJECT_NAME" "$file"
  fi
done
