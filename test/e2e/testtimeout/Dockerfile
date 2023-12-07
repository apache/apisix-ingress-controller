FROM golang:latest

WORKDIR /app

COPY . .

RUN go build -o gobackend .

EXPOSE 9280

ENTRYPOINT [ "/app/gobackend" ]