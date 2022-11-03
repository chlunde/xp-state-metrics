package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/crossplane-contrib/xp-state-metrics/metrics"

	"github.com/go-logr/logr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var (
		port       int
		kubeconfig *string
	)
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.IntVar(&port, "port", 8080, "Listen port number")
	flag.Parse()

	logr, _ := initLogging()

	cfg, err := restConfig(*kubeconfig)
	if err != nil {
		logr.Error(err, "could not get config")
		os.Exit(1)
	}

	// Grab a dynamic interface that we can create informers from
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		logr.Error(err, "could not generate dynamic client for config")
		os.Exit(1)
	}

	http.HandleFunc("/health/", func(http.ResponseWriter, *http.Request) {})

	handler, err := metrics.RunCollectors(context.Background(), dc)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/metrics", handler)

	logr.Info("started server", "port", port)
	go log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), loggingMiddleware(http.DefaultServeMux, logr)))

	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	close(stopCh)
}

func restConfig(kubeconfig string) (*rest.Config, error) {
	kubeCfg, err := rest.InClusterConfig()
	if err != nil {
		kubeCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	kubeCfg.QPS = 10
	kubeCfg.Burst = 15 // Should probably be 2x configured resource count (list + watch calls)
	return kubeCfg, nil
}

func loggingMiddleware(next http.Handler, log logr.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.Header.Get("x-forwarded-for")
		if remoteAddr == "" {
			remoteAddr = r.RemoteAddr
		}

		l := log.WithValues("path", r.URL.Path, "remoteAddr",
			remoteAddr, "x-request-id", r.Header.Get("x-request-id"))
		logContext := logr.NewContext(r.Context(), l)

		next.ServeHTTP(w, r.WithContext(logContext))
	})
}
