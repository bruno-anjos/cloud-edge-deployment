FROM alpine:latest AS builder

RUN apk add make git ncurses-terminfo-base ncurses-libs autoconf automake g++
RUN git clone https://github.com/vgropp/bwm-ng.git
WORKDIR bwm-ng
RUN ./autogen.sh && make && make install


FROM docker:19.03-dind

COPY --from=builder /usr/local/bin/bwm-ng /usr/local/bin/
RUN echo "@edge http://dl-cdn.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories && apk add iproute2-tc@edge

COPY deploy_services.sh /
COPY start_recording.sh /

# Client
COPY deployer-cli /
COPY archimedes-cli /
COPY deployments/ /deployments

# Deployer image dependencies
COPY fallback.json /

COPY load_images.sh /

RUN chmod +x deploy_services.sh start_recording.sh deployer-cli archimedes-cli load_images.sh

COPY setupTc.sh /setupTc.sh
COPY latency_map /latency_map
COPY ips_map /ips_map
COPY clean_dummy.sh /clean_dummy.sh
COPY setup_tc.sh /setup_tc.sh
RUN chmod +x /clean_dummy.sh /setup_tc.sh /setupTc.sh

ENTRYPOINT ["dockerd-entrypoint.sh"]