package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/targc/ktrack"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	tracker, err := ktrack.New(cfg, 15*time.Second, []ktrack.TrackItem{
		{
			APIGroup:  "apps",
			Kind:      "Deployment",
			Namespace: "default",
		},
		{
			APIGroup: "",
			Kind:     "Pod",
			Labels:   map[string]string{"ktrack": "true"},
		},
		{
			APIGroup:  "batch",
			Kind:      "Job",
			Namespace: "jobs",
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create tracker: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("tracking resources every 15s — press Ctrl+C to stop")

	tracker.Run(ctx, func(r ktrack.Resource) error {
		fmt.Printf("%-12s %-40s %v\n",
			r.Kind,
			r.Namespace+"/"+r.Name,
			r.Status["phase"],
		)
		return nil
	})
}

func loadConfig() (*rest.Config, error) {
	// In-cluster when running inside a pod.
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	// Fall back to ~/.kube/config for local dev.
	return clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
}
