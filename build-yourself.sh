#!/bin/bash

count=${1:?Usage: build-yourself.sh <count>}

for ((i = 1; i <= count; i++)); do
  go build -o skinner ./cmd/skinner && ./skinner PROMPT_BUILD.md 1
done
