package k8s

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func GetCurrentNodeName(clientset *kubernetes.Clientset) (string, error) {

	nodeName := os.Getenv("MY_NODE_NAME")
	if nodeName == "" {
		podName := os.Getenv("MY_POD_NAME")
		podNamespace := os.Getenv("MY_POD_NAMESPACE")
		if podName == "" || podNamespace == "" {
			return "", errors.Errorf("environeent variables MY_NODE_NAME and MY_POD_NAME are not set")
		}
		pod, err := clientset.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return "", errors.Errorf("failed to get pod %s in namespace %s: %s", podName, podNamespace, err.Error())
		}
		nodeName = pod.Spec.NodeName
		if nodeName == "" {
			return "", errors.Errorf("pod %s in namespace %s does not have a node name set", podName, podNamespace)
		}
	}

	return os.Getenv("MY_NODE_NAME"), nil
}

func GetKubeClientSet() *kubernetes.Clientset {

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Failed to build in cluster config: %s", err.Error())
	}

	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error building kubernetes clientset: %s", err.Error())
	}
	return kubeClientset

}

func InitKubeConfig() (*rest.Config, error) {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Failed to build config from flags: %s, using in cluster config", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("Failed to build in cluster config: %s", err.Error())
			return nil, err
		}
	}
	return config, nil

}

func GetNodeCIDR() string{
	nodeList := GetNodes()
	currentNodeName, err := GetCurrentNodeName(GetKubeClientSet())
	currentNodeCIDR := ""
	if err != nil{

	}
	for _, node := range nodeList.Items{

		if currentNodeName == node.Name{
			currentNodeCIDR = node.Spec.PodCIDR
		}
	}
	return currentNodeCIDR

}

func GetNodes()*v1.NodeList{

	nodes, err:=GetKubeClientSet().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {

		log.Print("Not able to retrieve ")
	}
	return nodes
}
