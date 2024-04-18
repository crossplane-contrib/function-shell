package main

import (
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var clientsetObtained bool
var clientsetGlobal *kubernetes.Clientset

func inClusterClient() (*kubernetes.Clientset, error) {
	if clientsetObtained {
		return clientsetGlobal, nil
	}

	var clientset *kubernetes.Clientset

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return clientset, err
	}

	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return clientset, err
	}

	clientsetObtained = true
	clientsetGlobal = clientset
	return clientset, nil
}

func outOfClusterClient() (*kubernetes.Clientset, error) {
	if clientsetObtained {
		return clientsetGlobal, nil
	}

	var clientset *kubernetes.Clientset
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return clientset, err
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return clientset, err
	}

	clientsetObtained = true
	clientsetGlobal = clientset
	return clientset, nil
}
