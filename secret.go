package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/crossplane/function-shell/input/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var clientsetObtained bool
var clientsetGlobal *kubernetes.Clientset

func addShellEnvVarsFromSecret(secretRef v1beta1.ShellEnvVarsSecretRef, shellEnvVars map[string]string) (map[string]string, error) {
	var clientset *kubernetes.Clientset

	_, err := os.OpenFile("/var/run/secrets/kubernetes.io", os.O_RDWR, 0666)
	if os.IsNotExist(err) {
		clientset, err = outOfClusterClient()
		if err != nil {
			return shellEnvVars, err
		}
	} else {
		clientset, err = inClusterClient()
		if err != nil {
			return shellEnvVars, err
		}
	}

	secret, err := getSecret(clientset, secretRef.Name, secretRef.Namespace)
	if err != nil {
		return shellEnvVars, err
	}
	secretEnvVars, err := getSecretEnvVars(secret, secretRef.Key)
	if err != nil {
		return shellEnvVars, err
	}

	maps.Copy(shellEnvVars, secretEnvVars)
	return shellEnvVars, nil
}

func getSecretEnvVars(secret *v1.Secret, key string) (map[string]string, error) {
	var envVarsData map[string]string
	if err := json.Unmarshal(secret.Data[key], &envVarsData); err != nil {
		return map[string]string(nil), err
	}
	return envVarsData, nil
}

func getSecret(clientset *kubernetes.Clientset, name, namespace string) (*v1.Secret, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		err = fmt.Errorf("Secret %s in namespace %s not found\n", name, namespace)
		return secret, err
	}

	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		err = fmt.Errorf("Error getting secret %s in namespace %s: %v\n",
			name, namespace, statusError.ErrStatus.Message)
		return secret, err
	}

	if err != nil {
		return secret, err
	}

	return secret, nil
}

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
