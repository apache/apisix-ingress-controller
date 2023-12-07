FROM golang:latest

WORKDIR /app

COPY . .

RUN make build

EXPOSE 9280

ENTRYPOINT [ "/app/gobackend" ]