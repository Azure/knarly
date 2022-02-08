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
		// Namespaces indicates the number of namespaces to use for all pods
		Namespaces int
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

	StatefulSetTestConfig struct {
		// Namespaces indicates the number of namespaces to use for all pods
		Namespaces int
		// InstancesPerNamespace indicates the number of statefulset instances per namespace
		InstancesPerNamespace int
		// TotalScaleSteps is the number of scale steps to use during a test run
		TotalScaleSteps int
		// PodsPerScaleStep is the number of stateful set pods to schedule per step
		PodsPerScaleStep int
		// StepDelayMinutes is the delay in between scale steps, in minutes
		StepDelayMinutes int
		// PvcStorageClass declares which type of storage driver to use, valid values are 'azurefile-csi' or 'azuredisk-csi'
		PvcStorageClass string
		// PvcStorageQuantity is the amount of storage to reserve, e.g., '8Gi'
		PvcStorageQuantity string
		// PodManagementPolicy; choose 'OrderedReady' for incremental statefulset scale up, 'Parallel' for complete immediate pod creation
		PodManagementPolicy string
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
	clusterloader2Command.Env = append(os.Environ(), fmt.Sprintf("CL2_NS_COUNT=%d", testConfig.Namespaces),
		fmt.Sprintf("CL2_CLEANUP=%d", testConfig.Cleanup),
		fmt.Sprintf("CL2_REPEATS=%d", testConfig.NumChurnIterations),
		fmt.Sprintf("CL2_POD_START_TIMEOUT_MINS=%d", testConfig.PodStartTimeoutMins),
		fmt.Sprintf("CL2_PODS_PER_NODE=%d", testConfig.PodsPerNode),
		fmt.Sprintf("CL2_TARGET_POD_CHURN=%d", testConfig.PodChurnRate),
		fmt.Sprintf("CL2_PODS_PER_DEPLOYMENT=%d", testConfig.PodsPerDeployment))
	clusterloader2Command.Dir = gitRootFilepath
	fmt.Printf("clusterloader2Command: %#v\n", clusterloader2Command)
	out, err := clusterloader2Command.CombinedOutput()
	utils.Logf("%s\n", out)
	Expect(err).ToNot(HaveOccurred())
}

func RunStatefulSetTest(ctx context.Context, input ClusterTestInput, testConfig StatefulSetTestConfig) {
	specName := "run-stateful-set-files-tests"
	Expect(input.BootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
	Expect(input.Cluster).NotTo(BeNil(), "Invalid argument. input.Cluster can't be nil when calling %s spec", specName)
	clusterProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, input.Cluster.Namespace, input.Cluster.Name)
	clientSet := clusterProxy.GetClientSet()
	nodesList, err := clientSet.CoreV1().Nodes().List(ctx, v1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	numNodes := len(nodesList.Items)
	kubeConfigPath := clusterProxy.GetKubeconfigPath()
	ex, err := os.Executable()
	Expect(err).ToNot(HaveOccurred())
	cwd := filepath.Dir(ex)
	cwdSlice := strings.Split(cwd, "/")
	gitRootFilepath := strings.Join(cwdSlice[:len(cwdSlice)-2], "/")
	clusterloader2Command := exec.Command("perf-tests/clusterloader2/cmd/clusterloader", fmt.Sprintf("--testconfig=%s/test/workloads/incremental-scale/config.yaml", gitRootFilepath), "--provider=aks", fmt.Sprintf("--kubeconfig=%s", kubeConfigPath), "--v=2", "--enable-exec-service=false")
	clusterloader2Command.Env = append(os.Environ(), fmt.Sprintf("CL2_NS_COUNT=%d", testConfig.Namespaces),
		fmt.Sprintf("CL2_INSTANCES_PER_NS=%d", testConfig.InstancesPerNamespace),
		fmt.Sprintf("CL2_TOTAL_SCALE_STEPS=%d", testConfig.TotalScaleSteps),
		fmt.Sprintf("CL2_PODS_PER_SCALE_STEP=%d", numNodes),
		fmt.Sprintf("CL2_STEP_DELAY=%dm", testConfig.StepDelayMinutes),
		fmt.Sprintf("CL2_PVC_STORAGE_CLASS=%s", testConfig.PvcStorageClass),
		fmt.Sprintf("CL2_PVC_STORAGE_QUANTITY=%s", testConfig.PvcStorageQuantity),
		fmt.Sprintf("CL2_POD_MANAGEMENT_POLICY=%s", testConfig.PodManagementPolicy))
	clusterloader2Command.Dir = gitRootFilepath
	fmt.Printf("clusterloader2Command: %#v\n", clusterloader2Command)
	out, err := clusterloader2Command.CombinedOutput()
	utils.Logf("%s\n", out)
	Expect(err).ToNot(HaveOccurred())
}
