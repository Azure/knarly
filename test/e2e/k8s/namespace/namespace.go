package namespace

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Get returns a namespace for with a given name
func Get(ctx context.Context, clientset *kubernetes.Clientset, name string) (*corev1.Namespace, error) {
	opts := metav1.GetOptions{}
	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, opts)
	if err != nil {
		log.Printf("failed trying to get namespace (%s):%s\n", name, err.Error())
		return nil, err
	}

	return namespace, nil
}
