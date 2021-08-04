package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/azure/knarly/test/e2e/specs"
	"github.com/azure/knarly/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("Workload cluster creation", func() {
	var (
		ctx               = context.TODO()
		specName          = "create-workload-cluster"
		namespace         *corev1.Namespace
		cancelWatches     context.CancelFunc
		result            *clusterctl.ApplyClusterTemplateAndWaitResult
		clusterName       string
		clusterNamePrefix string
		additionalCleanup func()
		specTimes         = map[string]time.Time{}
	)

	BeforeEach(func() {
		utils.LogCheckpoint(specTimes)

		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		Expect(e2eConfig).ToNot(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersion))

		clusterNamePrefix = fmt.Sprintf("knarly-e2e-%s", util.RandomString(6))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		var err error
		namespace, cancelWatches, err = utils.SetupSpecNamespace(ctx, clusterNamePrefix, bootstrapClusterProxy, artifactFolder)
		Expect(err).NotTo(HaveOccurred())

		result = new(clusterctl.ApplyClusterTemplateAndWaitResult)

		spClientSecret := os.Getenv(utils.AzureClientSecret)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-identity-secret",
				Namespace: namespace.Name,
				Labels: map[string]string{
					clusterv1.ClusterctlMoveHierarchyLabelName: "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"clientSecret": []byte(spClientSecret)},
		}
		err = bootstrapClusterProxy.GetClient().Create(ctx, secret)
		Expect(err).ToNot(HaveOccurred())

		identityName := e2eConfig.GetVariable(utils.ClusterIdentityName)
		Expect(os.Setenv(utils.ClusterIdentityName, identityName)).NotTo(HaveOccurred())
		Expect(os.Setenv(utils.ClusterIdentityNamespace, namespace.Name)).NotTo(HaveOccurred())
		Expect(os.Setenv(utils.ClusterIdentitySecretName, "cluster-identity-secret")).NotTo(HaveOccurred())
		Expect(os.Setenv(utils.ClusterIdentitySecretNamespace, namespace.Name)).NotTo(HaveOccurred())
		additionalCleanup = nil
	})

	AfterEach(func() {
		if result.Cluster == nil {
			// this means the cluster failed to come up. We make an attempt to find the cluster to be able to fetch logs for the failed bootstrapping.
			_ = bootstrapClusterProxy.GetClient().Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace.Name}, result.Cluster)
		}

		cleanInput := utils.CleanupInput{
			SpecName:          specName,
			Cluster:           result.Cluster,
			ClusterProxy:      bootstrapClusterProxy,
			Namespace:         namespace,
			CancelWatches:     cancelWatches,
			IntervalsGetter:   e2eConfig.GetIntervals,
			SkipCleanup:       skipCleanup,
			AdditionalCleanup: additionalCleanup,
			ArtifactFolder:    artifactFolder,
			E2eConfig:         e2eConfig,
		}
		utils.DumpSpecResourcesAndCleanup(ctx, cleanInput)
		Expect(os.Unsetenv(utils.AzureResourceGroup)).NotTo(HaveOccurred())
		Expect(os.Unsetenv(utils.AzureVNetName)).NotTo(HaveOccurred())

		utils.LogCheckpoint(specTimes)
	})

	It("With the default flavor", func() {
		clusterName = utils.GetClusterName(clusterNamePrefix, "default")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   "default",
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eConfig.GetVariable(capi_e2e.KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, result)

		Context("Listing Namespaces in workload cluster", func() {
			specs.ListNamespaces(ctx, specs.ListNamespacesInput{
				BootstrapClusterProxy: bootstrapClusterProxy,
				Cluster:               result.Cluster,
			})
		})
	})

	It("With the aks flavor", func() {
		clusterName = utils.GetClusterName(clusterNamePrefix, "aks")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   "aks",
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eConfig.GetVariable(utils.AKSKubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
			ControlPlaneWaiters: clusterctl.ControlPlaneWaiters{
				WaitForControlPlaneInitialized:   WaitForControlPlaneInitialized,
				WaitForControlPlaneMachinesReady: WaitForControlPlaneMachinesReady,
			},
		}, result)

		Context("Listing Namespaces in workload cluster", func() {
			specs.ListNamespaces(ctx, specs.ListNamespacesInput{
				BootstrapClusterProxy: bootstrapClusterProxy,
				Cluster:               result.Cluster,
			})
		})
	})

})
