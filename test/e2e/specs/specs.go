package specs

import (
	"context"

	"github.com/azure/knarly/test/e2e/utils"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/test/framework"
)

type (
	ListNamespacesInput struct {
		BootstrapClusterProxy framework.ClusterProxy
		Cluster               *clusterv1.Cluster
	}
)

func ListNamespaces(ctx context.Context, input ListNamespacesInput) {
	specName := "list-namespaces"
	Expect(input.BootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
	Expect(input.Cluster).NotTo(BeNil(), "Invalid argument. input.Cluster can't be nil when calling %s spec", specName)
	clusterProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, input.Cluster.Namespace, input.Cluster.Name)
	clientSet := clusterProxy.GetClientSet()
	list, err := clientSet.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(list).ToNot(BeNil())
	utils.Logf("namespaces in workload cluster are %+v", list.Items)
}
