FROM busybox
COPY ./autonomic /autonomic
RUN mkdir /metrics
COPY ./metrics/* /metrics/

EXPOSE 50003

CMD ["/autonomic", "-d"]