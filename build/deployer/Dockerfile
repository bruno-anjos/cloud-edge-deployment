FROM frolvlad/alpine-glibc
COPY ./deployer /deployer
COPY ./fallback.txt /fallback.txt

EXPOSE 50002

CMD ["/deployer", "-d"]