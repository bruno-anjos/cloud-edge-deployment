FROM busybox:glibc

COPY ./dummy_client /dummy_client
COPY ./config.json /config.json

CMD ["/dummy_client", "-d", "-config", "config.json"]