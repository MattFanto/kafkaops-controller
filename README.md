# kafkaops-controller

A Kubernetes controller managing Kafka topic defined with a CustomResourceDefinition (CRD).

**Note:** go-get or vendor this package as `github.com/mattfanto/kafkaops-controller`.

## Details

Kafka Ops Controller is resource as code tool which allows you to automate the management of your Kafka topics from 
Kubernetes CRD, topics desired state can be version controlled via Kubernetes manifest file and applied with 
your existing tools and pipelines.


## Notes

This is just a POC for the moment don't use it in production

## Contrib

The update-codegen script will automatically generate the following files & directories:
```
pkg/apis/samplecontroller/v1alpha1/zz_generated.deepcopy.go
pkg/generated/
```
Changes should not be made to these files manually, and when creating your own controller based off of this implementation you should not copy these files and instead run the update-codegen script to generate your own.

As a temporary workaround to updated code gen I created a symbolic link from 
```shell
export WORKSPACE_DIR=$(pwd)
ln -s $WORKSPACE_DIR/kafkaops-controller $WORKSPACE_DIR/github.com/mattfanto/kafkaops-controller
```
so that generated file via `./hack/update-codegen.sh` are automatically sync in the right folder