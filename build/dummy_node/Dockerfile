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
COPY clean_dummy.sh /clean_dummy.sh
COPY setup_tc.sh /setup_tc.sh
RUN chmod +x /clean_dummy.sh /setup_tc.sh

RUN echo "@edge http://dl-cdn.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories && apk add iproute2-tc@edge

ENTRYPOINT ["dockerd-entrypoint.sh"]