package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubesystem  = "kube-system"
	activitylog = "azure-activity-logs"
)

type (
	AzureClusterProxy struct {
		framework.ClusterProxy
	}
	// myEventData is used to be able to Marshal insights.EventData into JSON
	// see https://github.com/Azure/azure-sdk-for-go/issues/8224#issuecomment-614777550
	myEventData insights.EventData
)

func NewAzureClusterProxy(name string, kubeconfigPath string, scheme *runtime.Scheme, options ...framework.Option) *AzureClusterProxy {
	proxy, ok := framework.NewClusterProxy(name, kubeconfigPath, scheme, options...).(framework.ClusterProxy)
	Expect(ok).To(BeTrue(), "framework.NewClusterProxy must implement capi_e2e.ClusterProxy")
	return &AzureClusterProxy{
		ClusterProxy: proxy,
	}
}

func (acp *AzureClusterProxy) CollectWorkloadClusterLogs(ctx context.Context, namespace, name, outputPath string) {
	Byf("Dumping workload cluster %s/%s logs", namespace, name)
	acp.ClusterProxy.CollectWorkloadClusterLogs(ctx, namespace, name, outputPath)

	aboveMachinesPath := strings.Replace(outputPath, "/machines", "", 1)

	Byf("Dumping workload cluster %s/%s kube-system pod logs", namespace, name)
	start := time.Now()
	acp.collectPodLogs(ctx, namespace, name, aboveMachinesPath)
	Byf("Fetching kube-system pod logs took %s", time.Since(start).String())

	Byf("Dumping workload cluster %s/%s Azure activity log", namespace, name)
	start = time.Now()
	acp.collectActivityLogs(ctx, aboveMachinesPath)
	Byf("Fetching activity logs took %s", time.Since(start).String())
}

func (acp *AzureClusterProxy) collectPodLogs(ctx context.Context, namespace string, name string, aboveMachinesPath string) {
	workload := acp.GetWorkloadCluster(ctx, namespace, name)
	pods := &corev1.PodList{}
	Expect(workload.GetClient().List(ctx, pods, client.InNamespace(kubesystem))).To(Succeed())

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			// Watch each container's logs in a goroutine so we can stream them all concurrently.
			go func(pod corev1.Pod, container corev1.Container) {
				defer GinkgoRecover()

				Byf("Creating log watcher for controller %s/%s, container %s", kubesystem, pod.Name, container.Name)
				logFile := path.Join(aboveMachinesPath, kubesystem, pod.Name, container.Name+".log")
				Expect(os.MkdirAll(filepath.Dir(logFile), 0755)).To(Succeed())

				f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					// Failing to fetch logs should not cause the test to fail
					Byf("Error opening file to write pod logs: %v", err)
					return
				}
				defer f.Close()

				opts := &corev1.PodLogOptions{
					Container: container.Name,
					Follow:    true,
				}

				podLogs, err := workload.GetClientSet().CoreV1().Pods(kubesystem).GetLogs(pod.Name, opts).Stream(ctx)
				if err != nil {
					// Failing to stream logs should not cause the test to fail
					Byf("Error starting logs stream for pod %s/%s, container %s: %v", kubesystem, pod.Name, container.Name, err)
					return
				}
				defer podLogs.Close()

				out := bufio.NewWriter(f)
				defer out.Flush()
				_, err = out.ReadFrom(podLogs)
				if err != nil && err != io.ErrUnexpectedEOF {
					// Failing to stream logs should not cause the test to fail
					Byf("Got error while streaming logs for pod %s/%s, container %s: %v", kubesystem, pod.Name, container.Name, err)
				}
			}(pod, container)
		}
	}
}

func (acp *AzureClusterProxy) collectActivityLogs(ctx context.Context, aboveMachinesPath string) {
	timeoutctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	settings, err := auth.GetSettingsFromEnvironment()
	Expect(err).NotTo(HaveOccurred())
	subscriptionID := settings.GetSubscriptionID()
	authorizer, err := settings.GetAuthorizer()
	Expect(err).NotTo(HaveOccurred())
	activityLogsClient := insights.NewActivityLogsClient(subscriptionID)
	activityLogsClient.Authorizer = authorizer

	groupName := os.Getenv(AzureResourceGroup)
	start := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	end := time.Now().UTC().Format(time.RFC3339)

	itr, err := activityLogsClient.ListComplete(timeoutctx, fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceGroupName eq '%s'", start, end, groupName), "")
	if err != nil {
		// Failing to fetch logs should not cause the test to fail
		Byf("Error fetching activity logs for resource group %s: %v", groupName, err)
		return
	}

	logFile := path.Join(aboveMachinesPath, activitylog, groupName+".log")
	Expect(os.MkdirAll(filepath.Dir(logFile), 0755)).To(Succeed())

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Failing to fetch logs should not cause the test to fail
		Byf("Error opening file to write activity logs: %v", err)
		return
	}
	defer f.Close()
	out := bufio.NewWriter(f)
	defer out.Flush()

	for ; itr.NotDone(); err = itr.NextWithContext(timeoutctx) {
		if err != nil {
			Byf("Got error while iterating over activity logs for resource group %s: %v", groupName, err)
			return
		}
		event := itr.Value()
		if to.String(event.Category.Value) != "Policy" {
			b, err := json.MarshalIndent(myEventData(event), "", "    ")
			if err != nil {
				Byf("Got error converting activity logs data to json: %v", err)
			}
			if _, err = out.WriteString(string(b) + "\n"); err != nil {
				Byf("Got error while writing activity logs for resource group %s: %v", groupName, err)
			}
		}
	}
}
