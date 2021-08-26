The Pod Churn test is about testing the rate of creating, starting and deleting pods.

As at 2021, guidance from the upstream K8s team is that, on a single API server, pod churn should be kept 
<= 50 operations per second (counting all pod creations + pod updates + pod deletes).
See https://kubernetes.slack.com/archives/C09QZTRH7/p1625582970127300?thread_ts=1625537180.126600&cid=C09QZTRH7.  BUT they also point out that you can achieve much higher churn rates with naked pods
AS WE ARE DOING HERE.

The parameters to the tests are a pod density figure (pods per node) and a target churn rate.  The test uses a loose heuristic 
(simply divide by 6), to convert that into a number of pods to create each second.  And it deletes the same number each second.
The remainder of the churn comes from update (patch) operations.

If you set a very high target churn - e.g. 1000, you'll typically see 
a burst of activity when pod objects are being created and deleted (high churn), and then a pause while the tests waits for the 
generate pods to actually start running (lower churn in this time period). To reduce or eliminate that waiting period
you need to increase the QPS limit for your `kube-scheduler`.

Note that the structure of the test is in a number of phases:

* Phase 1: make a bunch of pods (at the requested density). 
* Phase 2: (the heart of the test).  Delete existing pods and create new one, at approximately the target churn rate (as note above)
* Phase 3: cleanup - delete all the remaining pods

The pods don't do any work, their job in this test is simply to be created and destroyed.

See comments in the YAML files for further details.

The main file is config.yaml.  It refers to the other files.

