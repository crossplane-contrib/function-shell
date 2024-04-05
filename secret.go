package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"

	"github.com/crossplane-contrib/function-shell/input/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

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
