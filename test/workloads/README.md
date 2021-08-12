This directory contains test workloads. Specifically, YAML test specs in the ClusterLoader2 format.

Each subdirectory contains one test workload, and each test workload is designed to test one specific dimension of scalablilty - e.g. Pod Churn rate.

The tests get their target node count automatically via the `.Nodes` variable, which is automatically set by ClusterLoader2 to be the number of nodes in the cluster.

Other parameters, such as pod density, can be provided through environment variables or a ClusterLoader2 overrides file.

To run a test manually in ClusterLoader2, use a command line something like this
(PowerShell command shown, but it can easily be converted to Bash simply by changing removing all `$env:` and the semi-colons, and changing `USERPROFILE` to `~`)

```
$env:CLUSTERLOADERROOT="<pathToClusterLoader2Binary>"; $env:CL2_PODS_PER_NODE=12; $env:CL2_TARGET_POD_CHURN=200; ./clusterloader --testconfig<path>/pod-churn/config.yaml --provider=aks --kubeconfig=$env:USERPROFILE/.kube/config --v=2 --enable-exec-service=false --delete-stale-namespaces
```