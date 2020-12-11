package kubernetes

import (
	"context"
	"fmt"

	"github.com/tinyzimmer/k3p/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is a kubernetes client abstraction for k3s management operations.
type Client interface {
	GetNodeByIP(ip string) (*corev1.Node, error)
	GetIPByNodeName(name string) (string, error)
	ListNodes() ([]corev1.Node, error)
	RemoveNode(name string) error
}

// New returns a new Client for the k3s cluster using the given kubeconfig bytes
func New(cfg []byte) (Client, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(cfg)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &client{clientset}, nil
}

type client struct {
	clientset *kubernetes.Clientset
}

func (c *client) GetIPByNodeName(name string) (string, error) {
	node, err := c.clientset.
		CoreV1().
		Nodes().
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	labels := node.GetLabels()
	if ip, ok := labels[types.K3sInternalIPLabel]; ok {
		return ip, nil
	}
	return "", fmt.Errorf("Node %q is missing k3s internal IP label", name)
}

func (c *client) GetNodeByIP(ip string) (*corev1.Node, error) {
	nodeList, err := c.clientset.
		CoreV1().
		Nodes().
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", types.K3sInternalIPLabel, ip),
		})
	if err != nil {
		return nil, err
	}
	if len(nodeList.Items) == 0 {
		return nil, fmt.Errorf("No node with the IP %q found in the cluster", ip)
	}
	return &nodeList.Items[0], nil
}

func (c *client) ListNodes() ([]corev1.Node, error) {
	nodeList, err := c.clientset.
		CoreV1().
		Nodes().
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (c *client) RemoveNode(name string) error {
	return c.clientset.
		CoreV1().
		Nodes().
		Delete(context.TODO(), name, metav1.DeleteOptions{})
}
