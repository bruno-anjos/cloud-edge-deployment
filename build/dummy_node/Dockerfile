FROM dind:latest

COPY ./archimedes /archimedes
COPY ./deployer /deployer
COPY ./autonomic /autonomic
COPY ./scheduler /scheduler
COPY ./deployer-cli /deployer-cli

COPY ./fallback.txt /fallback.txt

RUN mkdir /metrics
COPY ./metrics/* /metrics/

COPY ./deployments /deployments

COPY ./run_all.sh /run_all.sh
RUN chmod +x /run_all.sh

RUN mkdir /logs

CMD sh /run_all.sh