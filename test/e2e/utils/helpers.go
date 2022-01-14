package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/azure/knarly/test/e2e/k8s/namespace"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api-provider-azure/azure"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

func Byf(format string, a ...interface{}) {
	By(fmt.Sprintf(format, a...))
}

// Logf prints info logs with a timestamp and formatting.
func Logf(format string, args ...interface{}) {
	log("INFO", format, args...)
}

// Log prints info logs with a timestamp.
func Log(message string) {
	log("INFO", message)
}

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

// ExecOnHost runs the specified command directly on a node's host, using an SSH connection
// proxied through a control plane host.
func ExecOnHost(controlPlaneEndpoint, hostname, port string, f io.StringWriter, command string,
	args ...string) error {
	sshConfig, err := newSSHConfig()
	if err != nil {
		return err
	}

	// Init a client connection to a control plane node via the public load balancer
	lbClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", controlPlaneEndpoint, port), sshConfig)
	if err != nil {
		return errors.Wrapf(err, "dialing public load balancer at %s", controlPlaneEndpoint)
	}

	// Init a connection from the control plane to the target node
	c, err := lbClient.Dial("tcp", fmt.Sprintf("%s:%s", hostname, port))
	if err != nil {
		return errors.Wrapf(err, "dialing from control plane to target node at %s", hostname)
	}

	// Establish an authenticated SSH conn over the client -> control plane -> target transport
	conn, chans, reqs, err := ssh.NewClientConn(c, hostname, sshConfig)
	if err != nil {
		return errors.Wrap(err, "getting a new SSH client connection")
	}
	client := ssh.NewClient(conn, chans, reqs)
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "opening SSH session")
	}
	defer session.Close()

	// Run the command and write the captured stdout to the file
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	if len(args) > 0 {
		command += " " + strings.Join(args, " ")
	}
	if err = session.Run(command); err != nil {
		return errors.Wrapf(err, "running command \"%s\"", command)
	}
	if _, err = f.WriteString(stdoutBuf.String()); err != nil {
		return errors.Wrap(err, "writing output to file")
	}

	return nil
}

func newSSHConfig() (*ssh.ClientConfig, error) {
	// find private key file used for e2e workload cluster
	keyfile := os.Getenv("AZURE_SSH_PUBLIC_KEY_FILE")
	if len(keyfile) > 4 && strings.HasSuffix(keyfile, "pub") {
		keyfile = keyfile[:(len(keyfile) - 4)]
	}
	if keyfile == "" {
		keyfile = ".sshkey"
	}
	if _, err := os.Stat(keyfile); os.IsNotExist(err) {
		if !filepath.IsAbs(keyfile) {
			// current working directory may be test/e2e, so look in the project root
			keyfile = filepath.Join("..", "..", keyfile)
		}
	}

	pubkey, err := publicKeyFile(keyfile)
	if err != nil {
		return nil, err
	}
	sshConfig := ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		User:            azure.DefaultUserName,
		Auth:            []ssh.AuthMethod{pubkey},
	}
	return &sshConfig, nil
}

// publicKeyFile parses and returns the public key from the specified private key file.
func publicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// FileOnHost creates the specified path, including parent directories if needed.
func FileOnHost(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return nil, err
	}
	return os.Create(path)
}

// LogCheckpoint prints a message indicating the start or end of the current test spec,
// including which Ginkgo node it's running on.
//
// Example output:
//   INFO: "With 1 worker node" started at Tue, 22 Sep 2020 13:19:08 PDT on Ginkgo node 2 of 3
//   INFO: "With 1 worker node" ran for 18m34s on Ginkgo node 2 of 3
func LogCheckpoint(specTimes map[string]time.Time) {
	text := CurrentGinkgoTestDescription().TestText
	start, started := specTimes[text]
	if !started {
		start = time.Now()
		specTimes[text] = start
		Logf("INFO: \"%s\" started at %s on Ginkgo node %d of %d\n", text,
			start.Format(time.RFC1123), GinkgoParallelNode(), config.GinkgoConfig.ParallelTotal)
	} else {
		elapsed := time.Since(start)
		Logf("INFO: \"%s\" ran for %s on Ginkgo node %d of %d\n", text,
			elapsed.Round(time.Second), GinkgoParallelNode(), config.GinkgoConfig.ParallelTotal)
	}
}

type CleanupInput struct {
	SpecName          string
	ClusterProxy      framework.ClusterProxy
	ArtifactFolder    string
	Namespace         *corev1.Namespace
	CancelWatches     context.CancelFunc
	Cluster           *clusterv1.Cluster
	IntervalsGetter   func(spec, key string) []interface{}
	SkipCleanup       bool
	AdditionalCleanup func()
	E2eConfig         *clusterctl.E2EConfig
}

func DumpSpecResourcesAndCleanup(ctx context.Context, input CleanupInput) {
	defer func() {
		input.CancelWatches()
		redactLogs(input.E2eConfig)
	}()

	if input.Cluster == nil {
		By("Unable to dump workload cluster logs as the cluster is nil")
	} else {
		Byf("Dumping logs from the %q workload cluster", input.Cluster.Name)
		input.ClusterProxy.CollectWorkloadClusterLogs(ctx, input.Cluster.Namespace, input.Cluster.Name, filepath.Join(input.ArtifactFolder, "clusters", input.Cluster.Name))
	}

	Byf("Dumping all the Cluster API resources in the %q namespace", input.Namespace.Name)
	// Dump all Cluster API related resources to artifacts before deleting them.
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    input.ClusterProxy.GetClient(),
		Namespace: input.Namespace.Name,
		LogPath:   filepath.Join(input.ArtifactFolder, "clusters", input.ClusterProxy.GetName(), "resources"),
	})

	if input.SkipCleanup {
		return
	}

	Byf("Deleting all clusters in the %s namespace", input.Namespace.Name)
	// While https://github.com/kubernetes-sigs/cluster-api/issues/2955 is addressed in future iterations, there is a chance
	// that cluster variable is not set even if the cluster exists, so we are calling DeleteAllClustersAndWait
	// instead of DeleteClusterAndWait
	framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
		Client:    input.ClusterProxy.GetClient(),
		Namespace: input.Namespace.Name,
	}, input.IntervalsGetter(input.SpecName, "wait-delete-cluster")...)

	Byf("Deleting namespace used for hosting the %q test spec", input.SpecName)
	framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
		Deleter: input.ClusterProxy.GetClient(),
		Name:    input.Namespace.Name,
	})

	if input.AdditionalCleanup != nil {
		Byf("Running additional cleanup for the %q test spec", input.SpecName)
		input.AdditionalCleanup()
	}

	Byf("Checking if any resources are left over in Azure for spec %q", input.SpecName)
	ExpectResourceGroupToBe404(ctx)
}

// ExpectResourceGroupToBe404 performs a GET request to Azure to determine if the cluster resource group still exists.
// If it does still exist, it means the cluster was not deleted and is leaking Azure resources.
func ExpectResourceGroupToBe404(ctx context.Context) {
	settings, err := auth.GetSettingsFromEnvironment()
	Expect(err).NotTo(HaveOccurred())
	subscriptionID := settings.GetSubscriptionID()
	authorizer, err := settings.GetAuthorizer()
	Expect(err).NotTo(HaveOccurred())
	groupsClient := resources.NewGroupsClient(subscriptionID)
	groupsClient.Authorizer = authorizer
	_, err = groupsClient.Get(ctx, os.Getenv(AzureResourceGroup))
	Expect(azure.ResourceNotFound(err)).To(BeTrue(), "The resource group in Azure still exists. After deleting the cluster all of the Azure resources should also be deleted.")
}

func SetupSpecNamespace(ctx context.Context, namespaceName string, clusterProxy framework.ClusterProxy, artifactFolder string) (*corev1.Namespace, context.CancelFunc, error) {
	Byf("Creating namespace %q for hosting the cluster", namespaceName)
	Logf("starting to create namespace for hosting the %q test spec", namespaceName)
	logPath := filepath.Join(artifactFolder, "clusters", clusterProxy.GetName())
	ns, err := namespace.Get(ctx, clusterProxy.GetClientSet(), namespaceName)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, nil, err
	}

	// namespace exists wire it up
	if err == nil {
		Byf("Creating event watcher for existing namespace %q", ns.Name)
		watchesCtx, cancelWatches := context.WithCancel(ctx)
		go func() {
			defer GinkgoRecover()
			framework.WatchNamespaceEvents(watchesCtx, framework.WatchNamespaceEventsInput{
				ClientSet: clusterProxy.GetClientSet(),
				Name:      ns.Name,
				LogFolder: logPath,
			})
		}()

		return ns, cancelWatches, nil
	}

	// create and wire up namespace
	ns, cancelWatches := framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
		Creator:   clusterProxy.GetClient(),
		ClientSet: clusterProxy.GetClientSet(),
		Name:      namespaceName,
		LogFolder: logPath,
	})

	return ns, cancelWatches, nil
}

func redactLogs(e2eConfig *clusterctl.E2EConfig) {
	By("Redacting sensitive information from logs")
	Expect(e2eConfig.Variables).To(HaveKey(RedactLogScriptPath))
	cmd := exec.Command(e2eConfig.GetVariable(RedactLogScriptPath))
	cmd.Run()
}

// GetClusterName gets the cluster name for the test cluster
// and sets the environment variables that depend on it.
func GetClusterName(prefix, specName string) string {
	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		clusterName = fmt.Sprintf("%s-%s", prefix, specName)
	}

	Logf("INFO: Cluster name is %s\n", clusterName)
	Expect(os.Setenv(AzureResourceGroup, clusterName)).NotTo(HaveOccurred())
	Expect(os.Setenv(AzureVNetName, fmt.Sprintf("%s-vnet", clusterName))).NotTo(HaveOccurred())
	return clusterName
}
