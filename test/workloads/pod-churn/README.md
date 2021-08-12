The Pod Churn test is about testing the rate of creating, starting and deleting pods.

As at 2021, guidance from the upstream K8s team is that, on a single API server, pod churn should be kept 
<= 50 operations per second (counting all pod creations + pod updates + pod deletes).
See https://kubernetes.slack.com/archives/C09QZTRH7/p1625582970127300?thread_ts=1625537180.126600&cid=C09QZTRH7

The parameters to the tests are a pod density figure (pods per node) and a target churn rate.  The test uses a loose heuristic 
(simply divide by 4), to convert that into a number of pods to create each second.  And it deletes the same number each second.
The remainder of the churn comes from update (patch) operations.

You can, if you wish, set a very high target churn - e.g. 1000. That figure will not be reached, but it basically means
"try and create and delete the pods as fast as you can".  However, with figures like that, what you'll see is 
a burst of activity when pod objects are being created and deleted (high churn), and then a pause while the tests waits for the 
generate pods to actually start running (lower churn in this time period). Whereas if you set a target roughly consistent with
50 per API server instance, you'll see a steadier rate without such peaks and troughs.

Note that the structure of the test is in a number of phases:

* Phase 0: make a bunch of pods (at the requested density). This is not governed by the target churn, we just try to do it as quick as possible
* Phase 1: (first real test phase).  Delete existing pods and create new one, at approximately the target churn rate (as note above)
* Phase 2: Repeat of phase 1. The repeat is to ensure we get consistent figures.
* Phase 3: cleanup - delete all the remaining pods

The pods don't do any work, their job in this test is simply to be created and destroyed.

See comments in the YAML files for further details.

The main file is config.yaml.  It refers to the other files.

