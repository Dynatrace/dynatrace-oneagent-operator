# Dynatrace OneAgent Operator

## Hack on Dynatrace OneAgent Operator

[Operator SDK](https://github.com/operator-framework/operator-sdk) is the underlying framework for Dynatrace
OneAgent Operator. The `operator-sdk` tool needs to be installed upfront as outlined in the
[Operator SDK User Guide](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli).

#### Build and push your image
Replace `REGISTRY` with your Registry\`s URN:
```
$ cd $GOPATH/src/github.com/Dynatrace/dynatrace-oneagent-operator
$ operator-sdk build REGISTRY/dynatrace-oneagent-operator
$ docker push REGISTRY/dynatrace-oneagent-operator
```


#### Deploy operator
Change the `image` field in `./deploy/operator.yaml` to the URN of your image.
Apart from that follow the instructions in the usage section above.
