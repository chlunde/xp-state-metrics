package metrics

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
)

type InfoMapping struct {
	Label string

	Fieldpath string
	Getter    func(obj any) string
}

func xpCollector(name string, gvr schema.GroupVersionResource, store *cache.Store, mappings ...InfoMapping) collector {
	return (&XPCollector{name: name, gvr: gvr, mappings: mappings, store: store})
}

type XPCollector struct {
	name               string
	gvr                schema.GroupVersionResource
	disableReadySynced bool
	mappings           []InfoMapping
	store              *cache.Store
}

func (x *XPCollector) Run(ctx context.Context, client dynamic.Interface) *metricsstore.MetricsStore {
	headers := []string{
		`# TYPE %s gauge
# HELP %s A metrics series for each object`,
		`# TYPE %s_created gauge
# HELP %s_created Unix creation timestamp`,
		`# TYPE %s_labels gauge
# HELP %s_labels Labels from the kubernetes object`,
	}

	if len(x.mappings) > 0 {
		headers = append(headers, `# TYPE %s_info gauge
# HELP %s_info A metrics series exposing parameters as labels`)
	}

	if !x.disableReadySynced {
		headers = append(headers,
			`# TYPE %s_ready gauge
# HELP %s_ready A metrics series mapping the Ready status condition to a value (True=1,False=0,other=-1)`,
			`# TYPE %s_ready_time gauge
# HELP %s_ready_time Unix timestamp of last ready change`,
			`# TYPE %s_synced gauge
# HELP %s_synced A metrics series mapping the Synced status condition to a value (True=1,False=0,other=-1)`,
			`# TYPE %s_synced_time gauge
# HELP %s_synced_time Unix timestamp of last synced change`)
	}

	for i, hfmt := range headers {
		headers[i] = fmt.Sprintf(hfmt, x.name, x.name)
	}

	var block sync.Mutex

	var reflectorStore cache.Store
	metricsStore := metricsstore.NewMetricsStore(headers, x.objectMetrics)
	if x.store != nil { // set to non-nil if we should make a cache usable from other code
		reflectorStore = NewTeeStore(*x.store, metricsStore)
		block.Lock()
	} else {
		reflectorStore = metricsStore
	}

	lw := cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			o, err := client.Resource(x.gvr).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Println(x.gvr)
			}
			if x.store != nil {
				block.Unlock()
			}
			return o, err
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return client.Resource(x.gvr).Namespace(metav1.NamespaceAll).Watch(ctx, opts)
		},
	}

	r := cache.NewReflector(&lw, &unstructured.Unstructured{}, reflectorStore, 0)

	go r.Run(ctx.Done())
	if x.store != nil {
		block.Lock()
	}

	return metricsStore
}

func safeLabel(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z':
			return r
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-',
			r == '_',
			r == '.',
			r == '/':
			return '_'
		}
		return -1
	}, s)
}

func (x *XPCollector) objectMetrics(objAny any) []metric.FamilyInterface {
	obj := objAny.(*unstructured.Unstructured)
	paved := fieldpath.Pave(obj.Object)

	namespace := obj.GetNamespace()

	// If it's not a namespaced object, get the claim namespace when possible
	if namespace == "" {
		namespace, _ = paved.GetString(`metadata.labels["crossplane.io/claim-namespace"]`)
	}

	o := metric.Family{
		Name: x.name,
		Metrics: []*metric.Metric{
			{
				LabelKeys:   []string{"namespace", "name"},
				LabelValues: []string{namespace, obj.GetName()},
				Value:       1,
			},
		},
	}

	created := metric.Family{
		Name: x.name + "_created",
		Metrics: []*metric.Metric{
			{
				LabelKeys:   []string{"namespace", "name"},
				LabelValues: []string{namespace, obj.GetName()},
				Value:       float64(obj.GetCreationTimestamp().Unix()),
			},
		},
	}
	families := []metric.FamilyInterface{&o, &created}

	labels := metric.Family{
		Name: x.name + "_labels",
		Metrics: []*metric.Metric{
			{
				LabelKeys:   []string{"namespace", "name"},
				LabelValues: []string{namespace, obj.GetName()},
				Value:       float64(1),
			},
		},
	}

	for k, v := range obj.GetLabels() {
		labels.Metrics[0].LabelKeys = append(labels.Metrics[0].LabelKeys, "label_"+safeLabel(k))
		labels.Metrics[0].LabelValues = append(labels.Metrics[0].LabelValues, v)
	}
	families = append(families, &labels)

	if len(x.mappings) > 0 {
		var infoHeaders, infoValues []string
		for _, m := range x.mappings {

			var val string
			if m.Getter != nil {
				val = m.Getter(objAny)
			} else {
				val, _ = paved.GetString(m.Fieldpath)
			}

			infoValues = append(infoValues, val)
			infoHeaders = append(infoHeaders, m.Label)
		}

		o_info := metric.Family{
			Name: x.name + "_info",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   append([]string{"namespace", "name"}, infoHeaders...),
					LabelValues: append([]string{namespace, obj.GetName()}, infoValues...),
					Value:       1,
				},
			},
		}
		families = append(families, &o_info)
	}

	if !x.disableReadySynced {
		status := getCrossplaneStatus(obj)

		o_ready := metric.Family{
			Name: x.name + "_ready",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "name"},
					LabelValues: []string{namespace, obj.GetName()},
					Value:       status.ready,
				},
			},
		}

		o_ready_time := metric.Family{
			Name: x.name + "_ready_time",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "name"},
					LabelValues: []string{namespace, obj.GetName()},
					Value:       float64(status.readyTime.Unix()),
				},
			},
		}

		o_synced := metric.Family{
			Name: x.name + "_synced",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "name"},
					LabelValues: []string{namespace, obj.GetName()},
					Value:       status.ready,
				},
			},
		}
		o_synced_time := metric.Family{
			Name: x.name + "_synced_time",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "name"},
					LabelValues: []string{namespace, obj.GetName()},
					Value:       float64(status.syncedTime.Unix()),
				},
			},
		}
		families = append(families, &o_ready, &o_ready_time, &o_synced, &o_synced_time)
	}

	return families
}
