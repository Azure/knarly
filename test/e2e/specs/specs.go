package specs

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/azure/knarly/test/e2e/utils"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/test/framework"
)

type (
	ClusterTestInput struct {
		BootstrapClusterProxy framework.ClusterProxy
		Cluster               *clusterv1.Cluster
	}
)

func ListNamespaces(ctx context.Context, input ClusterTestInput) {
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

func RunConformance(ctx context.Context, input ClusterTestInput) {
	specName := "run-conformance"
	Expect(input.BootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
	Expect(input.Cluster).NotTo(BeNil(), "Invalid argument. input.Cluster can't be nil when calling %s spec", specName)
	clusterProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, input.Cluster.Namespace, input.Cluster.Name)
	kubeConfigPath := clusterProxy.GetKubeconfigPath()
	sonobuoyCommand := exec.Command("sonobuoy", "run", "--wait", "--plugin", "e2e", "--wait", "--e2e-skip", "\\[Serial\\]", "--e2e-focus", "\\[Conformance\\]", "--timeout=86400")
	sonobuoyCommand.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath))
	out, err := sonobuoyCommand.CombinedOutput()
	Expect(err).ToNot(HaveOccurred())
	utils.Logf("%s\n", out)
}

func RunPodChurnTest(ctx context.Context, input ClusterTestInput) {
	specName := "run-pod-churn-tests"
	Expect(input.BootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
	Expect(input.Cluster).NotTo(BeNil(), "Invalid argument. input.Cluster can't be nil when calling %s spec", specName)
	clusterProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, input.Cluster.Namespace, input.Cluster.Name)
	kubeConfigPath := clusterProxy.GetKubeconfigPath()
	clusterloader2Command := exec.Command("go", "run", "cmd/clusterloader.go", "--testconfig=../../test/workloads/deployment-churn/config.yaml", "--provider=aks", fmt.Sprintf("--kubeconfig=%s", kubeConfigPath), "--v=2", "--enable-exec-service=false")
	clusterloader2Command.Env = append(os.Environ(), fmt.Sprintf("CL2_NS_COUNT=%d", 15),
		fmt.Sprintf("CL2_CLEANUP=%d", 0),
		fmt.Sprintf("CL2_REPEATS=%d", 4),
		fmt.Sprintf("CL2_POD_START_TIMEOUT_MINS=%d", 20),
		fmt.Sprintf("CL2_PODS_PER_NODE=%d", 5),
		fmt.Sprintf("CL2_TARGET_POD_CHURN=%d", 75),
		fmt.Sprintf("CL2_PODS_PER_DEPLOYMENT=%d", 32))
	clusterloader2Command.Dir = "perf-tests/clusterloader2"
	out, err := clusterloader2Command.CombinedOutput()
	Expect(err).ToNot(HaveOccurred())
	utils.Logf("%s\n", out)
}
