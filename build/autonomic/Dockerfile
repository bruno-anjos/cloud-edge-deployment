FROM frolvlad/alpine-glibc
COPY ./autonomic /autonomic
RUN mkdir /metrics
COPY ./metrics/* /metrics/
COPY ./node_ips.json /node_ips.json

EXPOSE 50003

CMD ["/autonomic", "-d"]