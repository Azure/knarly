This directory contains test workloads. Specifically, YAML test specs in the ClusterLoader2 format.

Each subdirectory contains one test workload, and each test workload is designed to test one specific dimension of scalablilty - e.g. Pod Churn rate.

The tests get their target node count automatically via the `.Nodes` variable, which is automatically set by ClusterLoader2 to be the number of nodes in the cluster.

Other parameters, such as pod density, can be provided through environment variables or a ClusterLoader2 overrides file.

To run a test manually in ClusterLoader2, use a command line something like this (note that the env vars shown here are specific to the deployment churn test)

Bash. Mimimal set of parameters
```
CL2_PODS_PER_NODE=6 CL2_TARGET_POD_CHURN=50 CL2_PODS_PER_DEPLOYMENT=20 ~/go/src/github.com/kubernetes/perf-tests/clusterloader2/clusterloader2 --testconfig=./test/workloads/deployment-churn/config.yaml --provider=aks --kubeconfig=$HOME/.kube/config --v=2 --enable-exec-service=false
```

# Running tests faster

To run tests faster, set up your api server to have its command line parameter `delete-collection-workers`
set to a relatively high figure (e.g. 250 instead of its default value of 1).  Then set these two 
parameters at the start of your commmand:

```
CL2_CLEANUP=0 CL2_CHURN_FRACTION=0.5
```

The cleanup parameter supresses explicit deletion of the created objects, and lets deletion
happen as part of the namespace deletion that happens automatically at the end of the test.
For non-trivial load tests, that is performant only if `delete-collection-workers` has been set 
as noted above.

The churn fraction parameter says what fraction of the running pods should be replaced in the 
"churn" phase of the test. By default, that fraction is 1.0, i.e. all of them.  But for tests 
with lots of pods, you might want something that runs quicker than the default. So you can use 
`0.5` or any other value between 0 and 1.