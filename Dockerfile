FROM golang:1.13.8 AS build-env
MAINTAINER gxthrj@163.com

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

FROM alpine:3.11
MAINTAINER gxthrj@163.com

RUN mkdir /root/ingress-controller \
   && apk update  \
   && apk add ca-certificates \
   && update-ca-certificates \
   && apk add --no-cache libc6-compat \
   && echo "hosts: files dns" > /etc/nsswitch.conf \
   && rm -rf /var/cache/apk/*


WORKDIR /root/ingress-controller
COPY --from=build-env /root/ingress-controller/* /root/ingress-controller/
COPY --from=build-env /usr/share/zoneinfo/Hongkong /etc/localtime
EXPOSE 8080
RUN chmod +x ./build.sh
CMD ["/root/ingress-controller/build.sh"]