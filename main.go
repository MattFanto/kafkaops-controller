package main

import (
	"flag"
	"github.com/mattfanto/kafkaops-controller/pkg/resources/kafkaops"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientset "github.com/mattfanto/kafkaops-controller/pkg/generated/clientset/versioned"
	informers "github.com/mattfanto/kafkaops-controller/pkg/generated/informers/externalversions"
	"github.com/mattfanto/kafkaops-controller/pkg/signals"
)

var (
	masterURL       string
	kubeconfig      string
	bootstrapSevers string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	exampleInformerFactory := informers.NewSharedInformerFactory(exampleClient, time.Second*30)

	kafkaSdk, err := kafkaops.NewKafkaSdk(bootstrapSevers)
	if err != nil {
		panic(err)
	}
	controller := NewController(
		kubeClient,
		exampleClient,
		//kubeInformerFactory.Apps().V1().Deployments(),
		exampleInformerFactory.Kafkaopscontroller().V1alpha1().KafkaTopics(),
		kafkaSdk,
	)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	exampleInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	// TODO maybe a good idea to remove this one and specify the bootstrap servers in KafkaTopic CRD to allow management of multiple severs
	flag.StringVar(&bootstrapSevers, "bootstrap-servers", "", "Bootstrap servers list")
}
