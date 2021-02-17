FROM frolvlad/alpine-glibc
COPY ./autonomic /autonomic
COPY ./node_ips.json /node_ips.json

EXPOSE 50003

CMD ["/autonomic", "-d"]