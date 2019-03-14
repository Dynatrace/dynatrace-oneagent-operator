FROM registry.access.redhat.com/rhel-atomic

MAINTAINER Dynatrace

LABEL name="Dynatrace OneAgent Operator" \
      vendor="Dynatrace" \
      version="1.x" \
      release="1" \
      url="https://www.dynatrace.com" \
      summary="Dynatrace is an all-in-one, zero-config monitoring platform designed by and for cloud natives. It is powered by artificial intelligence that identifies performance problems and pinpoints their root causes in seconds." \
      description="Dynatrace OneAgent automatically discovers all technologies, services and applications that run on your host."

COPY LICENSE /licenses/

ADD build/_output/bin/dynatrace-oneagent-operator /usr/local/bin/dynatrace-oneagent-operator

USER 1001:1001
