echo -n `docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}}{{end}}' wolf-server`
