FROM alpine:3.10

RUN apk --no-cache add ca-certificates

USER 65534:65534

ADD build/_output/bin/dynatrace-oneagent-operator /usr/local/bin/dynatrace-oneagent-operator
