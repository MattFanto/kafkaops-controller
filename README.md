# kafkaops-controller

A Kubernetes controller managing Kafka topic defined with a CustomResourceDefinition (CRD).

**Note:** go-get or vendor this package as `github.com/mattfanto/kafkaops-controller`.

## Details

Kafka Ops Controller is resource as code tool which allows you to automate the management of your Kafka topics from 
Kubernetes CRD, topics desired state can be version controlled via Kubernetes manifest file and applied with 
your existing tools and pipelines.
