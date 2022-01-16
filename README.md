[![Continuous Integration](https://github.com/MattFanto/kafkaops-controller/actions/workflows/continuos-integration.yaml/badge.svg)](https://github.com/MattFanto/kafkaops-controller/actions/workflows/continuos-integration.yaml)
[![Continuous Delivery](https://github.com/MattFanto/kafkaops-controller/actions/workflows/continuous-delivery.yml/badge.svg)](https://github.com/MattFanto/kafkaops-controller/actions/workflows/continuous-delivery.yml)
[![Release Version](https://img.shields.io/github/v/release/MattFanto/kafkaops-controller?label=kafkaops-controller)](https://github.com/MattFanto/kafkaops-controller/releases/latest)

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

First install the CRD, deployment and roles definition available under [artifacts/deployment.yaml](artifacts/deployment.yaml), under a new kafkaops namespace
```shell
kubectl create namespace kafkaops
kubectl apply -n kafkaops -f https://github.com/MattFanto/kafkaops-controller/releases/latest/download/deployment.yaml
```


### Create a new KafkaTopic

Now let's create a `KafkaTopic` resource with the following YAML
```shell
cat << EOF > example_topic.yaml 
apiVersion: kafkaopscontroller.mattfanto.github.com/v1alpha1
kind: KafkaTopic
metadata:
  name: example-kafkatopic
spec:
  topicName: example_topic_v2
  replicas: 1
  partitions: 1
EOF
```

Apply the resource to your cluster
```shell
kubectl apply -n kafkaops -f example_topic.yaml
```

The topic will be created automatically in Kafka the kafkaops-controller if not exists, if it already
exists it will report any deviation from the original specification
```shell
kafka-topics.sh --list --bootstrap-server $BOOTSTRAP_SERVER
```

Topic status update will be reported will be reported in CRD Status section
```shell
kubectl describe kafkatopics.kafkaopscontroller.mattfanto.github.com -n kafkaops example-kafkatopic
```
will return
```yaml
...
Status:
  Conditions:
    Last Transition Time:  2022-01-15T16:55:49Z
    Message:               Topic ready and specification in sync
    Reason:                
    Status:                True
    Type:                  Ready
  Partitions:              1
  Replicas:                1
  Status Code:             EXISTS
Events:
  Type    Reason  Age                     From                 Message
  ----    ------  ----                    ----                 -------
  Normal  Synced  113s (x781 over 3h16m)  kafkaops-controller  KafkaTopic synced successfully
```

CLI example:

![gif](/docs/imgs/cli-example.gif)


### Cleanup

You can clean up the created CustomResourceDefinition with:
```shell
kubectl delete -n kafkaops -f artifacts/deployment.yaml
kubectl delete namespace kafkaops
```


## Usage with ARGO-CD

If you are using Argo-CD for your deployment, you will be able to visualize automatically KafkaTopic resource in ARGO cd, but if you also 
want ARGO to read and display the KafkaTopic status you need to add this Lua custom health check to your config [reference](https://argo-cd.readthedocs.io/en/stable/operator-manual/health/).
```yaml
argo-cd:
  server:
    config:
      resource.customizations: |-
        kafkaopscontroller.mattfanto.github.com/KafkaTopic:
          health.lua: |
            hs = {}
            if obj.status ~= nil then
              if obj.status.conditions ~= nil then
                for i, condition in ipairs(obj.status.conditions) do
                  if condition.type == "Ready" and condition.status == "False" then
                    hs.status = "Degraded"
                    hs.message = condition.message
                    return hs
                  end
                  if condition.type == "Ready" and condition.status == "True" then
                    hs.status = "Healthy"
                    hs.message = condition.message
                    return hs
                  end
                end
              end
            end

            hs.status = "Progressing"
            hs.message = "Waiting for certificate"
            return hs
```

After this you can see the KafkaTopic status in Argo-CD as an example, if the deviates from the original specification
you will see it as:

![diagram](/docs/imgs/argocd-integration.png)

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
