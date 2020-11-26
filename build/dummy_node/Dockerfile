FROM docker:19.03-dind

COPY deploy_services.sh /

# Client
COPY deployer-cli /
COPY deployments/ /deployments

# Autonomic image dependencies
COPY metrics/ /metrics

# Deployer image dependencies
COPY fallback.txt /

COPY load_images.sh /

COPY images/ /images

RUN chmod +x deploy_services.sh deployer-cli load_images.sh

ENTRYPOINT ["dockerd-entrypoint.sh"]