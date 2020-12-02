FROM golang:1.13.8 AS build-env
LABEL maintainer="gxthrj@163.com"


WORKDIR /go/src/github.com/api7/ingress-controller
COPY . .
RUN mkdir /root/ingress-controller \
    && go env -w GOPROXY=https://goproxy.io,direct \
    && export GOPROXY=https://goproxy.io \
    && go build -o /root/ingress-controller/ingress-controller \
    && mv /go/src/github.com/api7/ingress-controller/build.sh /root/ingress-controller/ \
    && mv /go/src/github.com/api7/ingress-controller/conf.json /root/ingress-controller/ \
    && rm -rf /go/src/github.com/api7/ingress-controller \
    && rm -rf /etc/localtime \
    && ln -s  /usr/share/zoneinfo/Hongkong /etc/localtime \
    && dpkg-reconfigure -f noninteractive tzdata

FROM alpine:3.12.1
LABEL maintainer="gxthrj@163.com"

RUN mkdir /root/ingress-controller \
   && apk add --no-cache ca-certificates libc6-compat \
   && update-ca-certificates \
   && echo "hosts: files dns" > /etc/nsswitch.conf


WORKDIR /root/ingress-controller
COPY --from=build-env /root/ingress-controller/* /root/ingress-controller/
COPY --from=build-env /usr/share/zoneinfo/Hongkong /etc/localtime
EXPOSE 8080
RUN chmod +x ./build.sh
CMD ["/root/ingress-controller/build.sh"]
