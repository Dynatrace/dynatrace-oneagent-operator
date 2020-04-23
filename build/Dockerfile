FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1

LABEL name="Dynatrace OneAgent Operator" \
      vendor="Dynatrace LLC" \
      maintainer="Dynatrace LLC" \
      version="1.x" \
      release="1" \
      url="https://www.dynatrace.com" \
      summary="Dynatrace is an all-in-one, zero-config monitoring platform designed by and for cloud natives. It is powered by artificial intelligence that identifies performance problems and pinpoints their root causes in seconds." \
      description="Dynatrace OneAgent automatically discovers all technologies, services and applications that run on your host."

ENV BIN_DIR=/usr/local/bin/dynatrace/oneagent \
    OPERATOR=${BIN_DIR}/operator \
    WEBHOOK_BOOTSTRAPPER=${BIN_DIR}/webhook-bootstrapper \
    WEBHOOK_SERVER=${BIN_DIR}/webhook-server \
    USER_UID=1001 \
    USER_NAME=dynatrace-oneagent-operator

RUN  microdnf install openssl unzip && microdnf clean all
COPY LICENSE /licenses/
COPY build/_output/bin ${BIN_DIR}
COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
