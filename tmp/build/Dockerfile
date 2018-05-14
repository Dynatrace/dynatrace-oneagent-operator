FROM alpine:3.6

ADD tmp/_output/bin/dynatrace-oneagent-operator /usr/local/bin/dynatrace-oneagent-operator

RUN apk -U add ca-certificates

RUN adduser -D dynatrace-oneagent-operator
USER dynatrace-oneagent-operator
