package metrics

import (
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type crossplaneStatus struct {
	ready      float64
	synced     float64
	readyTime  time.Time
	syncedTime time.Time
}

// statusToPrometheusValue maps a status to a float64 (True=1, False=0, Unknown/missing/other=-1). This allows us to use it as a metric value
func statusToPrometheusValue(s xpv1.ConditionedStatus, typ xpv1.ConditionType) float64 {
	switch s.GetCondition(typ).Status {
	case "True":
		return 1
	case "False":
		return 0
	default:
		return -1
	}
}

func getCrossplaneStatus(u *unstructured.Unstructured) crossplaneStatus {
	conditioned := xpv1.ConditionedStatus{}
	_ = fieldpath.Pave(u.Object).GetValueInto("status", &conditioned)

	return crossplaneStatus{
		ready:      statusToPrometheusValue(conditioned, xpv1.TypeReady),
		synced:     statusToPrometheusValue(conditioned, xpv1.TypeSynced),
		readyTime:  conditioned.GetCondition(xpv1.TypeReady).LastTransitionTime.Time,
		syncedTime: conditioned.GetCondition(xpv1.TypeSynced).LastTransitionTime.Time,
	}
}
