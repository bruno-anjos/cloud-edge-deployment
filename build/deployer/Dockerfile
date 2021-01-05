FROM frolvlad/alpine-glibc
COPY ./deployer /deployer
COPY ./fallback.json /fallback.json
COPY ./node_ips.json /node_ips.json

EXPOSE 50002

CMD ["/deployer", "-d"]