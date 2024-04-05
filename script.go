package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/crossplane-contrib/function-shell/input/v1beta1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func loadShellScripts(log logging.Logger, shellScriptsConfigMapsRef []v1beta1.ShellScriptsConfigMapRef) (map[string][]string, error) {
	shellScripts := make(map[string][]string)

	for _, shellScriptsConfigMapRef := range shellScriptsConfigMapsRef {
		newShellScripts, err := getShellScriptsFromConfigMap(shellScriptsConfigMapRef)
		if err != nil {
			log.Info("unable to get shell scripts from ", shellScriptsConfigMapRef.Name)
			continue
		}
		maps.Copy(shellScripts, newShellScripts)
	}

	return shellScripts, nil
}

func getShellScriptsFromConfigMap(shellScriptsConfigMapRef v1beta1.ShellScriptsConfigMapRef) (map[string][]string, error) {
	var clientset *kubernetes.Clientset
	scripts := make(map[string][]string)

	_, err := os.OpenFile("/var/run/secrets/kubernetes.io", os.O_RDWR, 0666)
	if os.IsNotExist(err) {
		clientset, err = outOfClusterClient()
		if err != nil {
			return scripts, err
		}
	} else {
		clientset, err = inClusterClient()
		if err != nil {
			return scripts, err
		}
	}

	scripts, err = getScripts(clientset, shellScriptsConfigMapRef)
	if err != nil {
		return scripts, err
	}

	return scripts, nil
}

func getScripts(clientset *kubernetes.Clientset, shellScriptsConfigMapRef v1beta1.ShellScriptsConfigMapRef) (map[string][]string, error) {
	scripts := make(map[string][]string)
	scriptNames := shellScriptsConfigMapRef.ScriptNames
	name := shellScriptsConfigMapRef.Name
	namespace := shellScriptsConfigMapRef.Namespace

	scriptConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		err = fmt.Errorf("Script ConfigMap %s in namespace %s not found\n", name, namespace)
		return scripts, err
	}

	if statusError, isStatus := err.(*errors.StatusError); isStatus {
		err = fmt.Errorf("Error getting script in ConfigMap %s in namespace %s: %v\n",
			name, namespace, statusError.ErrStatus.Message)
		return scripts, err
	}

	if err != nil {
		return scripts, err
	}

	for _, scriptName := range scriptNames {
		for _, scriptLine := range strings.Split(scriptConfigMap.Data[scriptName], "\n") {
			scripts[scriptName] = append(scripts[scriptName], scriptLine)
		}
	}

	return scripts, nil
}
