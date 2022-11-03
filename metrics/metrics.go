package metrics

import (
	"context"
	"io"
	"net/http"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
)

type MetricsHandler struct {
	mtx            sync.RWMutex
	metricsWriters []*metricsstore.MetricsStore
}

type collector interface {
	Run(ctx context.Context, client dynamic.Interface) *metricsstore.MetricsStore
}

/* demo for function getting something from another object

var (
	fooStore        cache.Store = cache.NewStore(cache.MetaNamespaceKeyFunc)
)

func fooGetter(objAny any) string {
	obj := objAny.(*unstructured.Unstructured)
    _, exists, _ := appNamespaceStore.GetByKey(....)
	return "unknown"
}
*/

func RunCollectors(ctx context.Context, client dynamic.Interface) (http.Handler, error) {
	// we could probably put this inside a yaml file / configmap
	collectors := []collector{
		/*
			xpCollector("pginstance",
				schema.GroupVersionResource{Group: "aws.company.example.com", Version: "v1alpha1", Resource: "pginstances"}, nil,
				InfoMapping{Getter: fooGetter, Label: "foo_x"},
				InfoMapping{Fieldpath: "spec.parameters.instanceType", Label: "instance_type"},
				InfoMapping{Fieldpath: "status.engineVersion", Label: "engine_version"},
				InfoMapping{Fieldpath: "status.dbResourceId", Label: "db_resource_id"},
			),
		*/
		xpCollector("rdsinstance",
			schema.GroupVersionResource{Group: "database.aws.crossplane.io", Version: "v1beta1", Resource: "rdsinstances"}, nil,
			InfoMapping{Fieldpath: "spec.forProvider.dbInstanceClass", Label: "instance_type"},
		),
	}

	var stores []*metricsstore.MetricsStore
	for _, c := range collectors {
		stores = append(stores, c.Run(ctx, client))
	}

	return &MetricsHandler{metricsWriters: stores}, nil
}

// ServeHTTP implements the http.Handler interface. It writes all generated
// metrics to the response body.
func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	resHeader := w.Header()
	var writer io.Writer = w

	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	for _, w := range m.metricsWriters {
		w.WriteAll(writer)
	}

	// In case we gzipped the response, we have to close the writer.
	if closer, ok := writer.(io.Closer); ok {
		closer.Close()
	}
}
