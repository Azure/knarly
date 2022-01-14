package specs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/azure/knarly/test/e2e/utils"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
)

type (
	ClusterTestInput struct {
		BootstrapClusterProxy framework.ClusterProxy
		Cluster               *clusterv1.Cluster
	}

	PodChurnTestConfig struct {
		// Cleanup indicates whether or not to explicitly cleanup pods after test, 0=no, 1=yes
		Cleanup int
		// NumChurnIterations is the number of pod churn lifecycles to perform during the test
		NumChurnIterations int
		// PodStartTimeoutMins indicates how long to wait for all pods to be running
		PodStartTimeoutMins int
		// PodsPerNode determines how many pods to create, per node, in the cluster
		PodsPerNode int
		// PodChurnRate configures the desired pods to create, and delete per second
		PodChurnRate int
		// PodsPerDeployment sets the maximum number of pod replicas in a single deployment; as more pods are needed,
		PodsPerDeployment int
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

func RunPodChurnTest(ctx context.Context, input ClusterTestInput, testConfig PodChurnTestConfig) {
	specName := "run-pod-churn-tests"
	Expect(input.BootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
	Expect(input.Cluster).NotTo(BeNil(), "Invalid argument. input.Cluster can't be nil when calling %s spec", specName)
	clusterProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, input.Cluster.Namespace, input.Cluster.Name)
	kubeConfigPath := clusterProxy.GetKubeconfigPath()
	ex, err := os.Executable()
	Expect(err).ToNot(HaveOccurred())
	cwd := filepath.Dir(ex)
	cwdSlice := strings.Split(cwd, "/")
	gitRootFilepath := strings.Join(cwdSlice[:len(cwdSlice)-2], "/")
	clusterloader2Command := exec.Command("perf-tests/clusterloader2/cmd/clusterloader", fmt.Sprintf("--testconfig=%s/test/workloads/deployment-churn/config.yaml", gitRootFilepath), "--provider=aks", fmt.Sprintf("--kubeconfig=%s", kubeConfigPath), "--v=2", "--enable-exec-service=false")
	clusterloader2Command.Env = append(os.Environ(), fmt.Sprintf("CL2_NS_COUNT=%d", 15),
		fmt.Sprintf("CL2_CLEANUP=%d", testConfig.Cleanup),
		fmt.Sprintf("CL2_REPEATS=%d", testConfig.NumChurnIterations),
		fmt.Sprintf("CL2_POD_START_TIMEOUT_MINS=%d", testConfig.PodStartTimeoutMins),
		fmt.Sprintf("CL2_PODS_PER_NODE=%d", testConfig.PodsPerNode),
		fmt.Sprintf("CL2_TARGET_POD_CHURN=%d", testConfig.PodChurnRate),
		fmt.Sprintf("CL2_PODS_PER_DEPLOYMENT=%d", testConfig.PodsPerDeployment))
	clusterloader2Command.Dir = gitRootFilepath
	out, err := clusterloader2Command.CombinedOutput()
	Expect(err).ToNot(HaveOccurred())
	utils.Logf("%s\n", out)
}
