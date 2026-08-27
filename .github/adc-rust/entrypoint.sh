#!/bin/sh
# The ingress deployment manifest (test/e2e/framework/manifests/ingress.yaml)
# invokes the sidecar as `server ...`, the Node adc CLI's subcommand name.
# The Rust adc CLI (as of the v0.30.0 release binary this image wraps) calls
# the same subcommand `ingress-server` instead, so translate here rather
# than touching the shared manifest.
set -e
if [ "$1" = "server" ]; then
  shift
  set -- ingress-server "$@"
fi
exec /usr/local/bin/adc "$@"
