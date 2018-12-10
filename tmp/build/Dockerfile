FROM alpine:3.6

ADD tmp/_output/bin/dynatrace-oneagent-operator /usr/local/bin/dynatrace-oneagent-operator

RUN apk -U add ca-certificates

RUN adduser -D -u 1001 dynatrace-oneagent-operator
USER 1001:1001
