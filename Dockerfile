FROM golang:1.17-alpine AS operator-build

RUN apk update --no-cache && \
    apk add --no-cache gcc musl-dev btrfs-progs-dev lvm2-dev device-mapper-static && \
    rm -rf /var/cache/apk/*

ARG GO_BUILD_ARGS
COPY . /app
WORKDIR /app

# move previously cached go modules to gopath
RUN if [ -d ./mod ]; then mkdir -p ${GOPATH}/pkg && [ -d mod ] && mv ./mod ${GOPATH}/pkg; fi;

RUN CGO_ENABLED=1 go build -ldflags "${GO_BUILD_ARGS}" -o ./build/_output/bin/dynatrace-oneagent-operator ./

################
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=operator-build /app/build/_output/bin /usr/local/bin

LABEL name="Dynatrace OneAgent Operator" \
      vendor="Dynatrace LLC" \
      maintainer="Dynatrace LLC" \
      version="1.x" \
      release="1" \
      url="https://www.dynatrace.com" \
      summary="Dynatrace is an all-in-one, zero-config monitoring platform designed by and for cloud natives. It is powered by artificial intelligence that identifies performance problems and pinpoints their root causes in seconds." \
      description="Dynatrace OneAgent automatically discovers all technologies, services and applications that run on your host."

ENV OPERATOR=dynatrace-oneagent-operator \
    USER_UID=1001 \
    USER_NAME=dynatrace-oneagent-operator

RUN  microdnf install unzip && microdnf clean all
COPY LICENSE /licenses/
COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
