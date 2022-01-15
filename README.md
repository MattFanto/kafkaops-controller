# kafkaops-controller

A Kubernetes controller managing Kafka topic defined with a CustomResourceDefinition (CRD).

![diagram](/docs/imgs/diagram.png)

**Note:** go-get or vendor this package as `github.com/mattfanto/kafkaops-controller`.

## Details

Kafka Ops Controller is resource as code tool which allows you to automate the management of your Kafka topics from 
Kubernetes CRD, topics desired state can be version controlled via Kubernetes manifest file and applied with 
your existing tools and pipelines (e.g. Helm, Argo-CD, ...).

Topics are defined and versioned as K8S manifest files (KafkaTopic CRD) and when applied to the cluster kafkaops-controller
listen for new topic or changes to apply the desired state to your Kafka cluster.

## Getting Started

### Installation 

First install the CRD, deployment and roles definition available under [artifacts/deployment.yaml](artifacts/deployment.yaml)
```shell
kubectl create namespace kafkaops
kubectl apply -n kafkaops -f artifacts/deployment.yaml
```

You can install a test topic and check the creation in kafka via
```shell
kubectl apply -n kafkaops -f artifacts/examples/example-topic.yaml
```


### Cleanup

You can clean up the created CustomResourceDefinition with:
```shell
kubectl delete -n kafkaops -f artifacts/deployment.yaml
kubectl delete namespace kafkaops
```

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
