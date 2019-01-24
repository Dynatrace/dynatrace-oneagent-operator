FROM alpine:3.8

RUN apk upgrade --update --no-cache
RUN apk add ca-certificates

USER 65534:65534

ADD build/_output/bin/dynatrace-oneagent-operator /usr/local/bin/dynatrace-oneagent-operator
