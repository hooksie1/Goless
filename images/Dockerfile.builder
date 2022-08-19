FROM golang:alpine

WORKDIR /data
RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN update-ca-certificates
ADD . /data/


CMD ["./build.sh"]