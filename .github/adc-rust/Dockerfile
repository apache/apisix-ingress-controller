# Wraps the Rust adc release binary (downloaded by the workflow into the
# build context as `adc`) into an image shaped enough like ghcr.io/api7/adc
# to stand in for it as the ingress sidecar in e2e tests.
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY adc /usr/local/bin/adc
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /usr/local/bin/adc /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
