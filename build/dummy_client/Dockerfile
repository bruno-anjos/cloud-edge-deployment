FROM frolvlad/alpine-glibc

RUN apk add --no-cache make gcc gawk bison linux-headers libc-dev

COPY ./dummy_client /dummy_client
COPY ./config.json /config.json

CMD ["/dummy_client", "-d", "-config", "config.json"]