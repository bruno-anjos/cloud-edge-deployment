FROM frolvlad/alpine-glibc
COPY ./deployer /deployer
COPY ./fallback.json /fallback.json

EXPOSE 50002

CMD ["/deployer", "-d"]