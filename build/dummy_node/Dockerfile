FROM docker:19.03-dind

COPY deploy_services.sh /

# Client
COPY deployer-cli /
COPY deployments/ /deployments

# Autonomic image dependencies
COPY metrics/ /metrics

# Deployer image dependencies
COPY fallback.json /

COPY load_images.sh /

RUN chmod +x deploy_services.sh deployer-cli load_images.sh

COPY setupTc.sh /setupTc.sh
COPY latency_map /latency_map
COPY ips_map /ips_map

RUN echo "@edge http://dl-cdn.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories && apk add iproute2-tc@edge

ENTRYPOINT ["dockerd-entrypoint.sh"]