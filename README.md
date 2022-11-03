# xp-state-metrics

POC/WIP for kube-state-metrics for crossplane, managed resources and even any third-party CRD

```
# TYPE rdsinstance gauge
# HELP rdsinstance A metrics series for each object
rdsinstance{namespace="application-b-ns",name="delta"} 1
rdsinstance{namespace="application-a-ns",name="alpha"} 1
# TYPE rdsinstance_created gauge
# HELP rdsinstance_created Unix creation timestamp
rdsinstance_created{namespace="application-b-ns",name="delta"} 1.666767144e+09
rdsinstance_created{namespace="application-a-ns",name="alpha"} 1.666700362e+09
# TYPE rdsinstance_labels gauge
# HELP rdsinstance_labels Labels from the kubernetes object
rdsinstance_labels{namespace="application-b-ns",name="delta",label_crossplane_io_claim_name="db",label_crossplane_io_claim_namespace="application-b-ns",label_crossplane_io_composite="db-h2bxt"} 1
rdsinstance_labels{namespace="application-a-ns",name="alpha",label_crossplane_io_claim_namespace="application-a-ns",label_crossplane_io_composite="foo-db-nmvsr",label_crossplane_io_claim_name="foo-db"} 1
# TYPE rdsinstance_info gauge
# HELP rdsinstance_info A metrics series exposing parameters as labels
rdsinstance_info{namespace="application-b-ns",name="delta",instance_type="db.t3.micro"} 1
rdsinstance_info{namespace="application-a-ns",name="alpha",instance_type="db.t3.small"} 1
# TYPE rdsinstance_ready gauge
# HELP rdsinstance_ready A metrics series mapping the Ready status condition to a value (True=1,False=0,other=-1)
rdsinstance_ready{namespace="application-b-ns",name="delta"} 1
rdsinstance_ready{namespace="application-a-ns",name="alpha"} -1
# TYPE rdsinstance_ready_time gauge
# HELP rdsinstance_ready_time Unix timestamp of last ready change
rdsinstance_ready_time{namespace="application-a-ns",name="alpha"} 0
rdsinstance_ready_time{namespace="application-b-ns",name="delta"} 1.666767438e+09
# TYPE rdsinstance_synced gauge
# HELP rdsinstance_synced A metrics series mapping the Synced status condition to a value (True=1,False=0,other=-1)
rdsinstance_synced{namespace="application-a-ns",name="alpha"} -1
rdsinstance_synced{namespace="application-b-ns",name="delta"} 1
# TYPE rdsinstance_synced_time gauge
# HELP rdsinstance_synced_time Unix timestamp of last synced change
rdsinstance_synced_time{namespace="application-a-ns",name="alpha"} 1.666700362e+09
rdsinstance_synced_time{namespace="application-b-ns",name="delta"} 1.666767499e+09
```

# Test

* Log in to kubernetes
* `go run .`
* `curl http://localhost:8080/metrics`
* Set up prometheus to scrape it
* Add managed resources and claim types to metrics/metrics.go
