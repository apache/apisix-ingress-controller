FROM scratch

COPY bin/e2e-echo-server /e2e-echo-server

EXPOSE 8080

ENTRYPOINT ["/e2e-echo-server"]
