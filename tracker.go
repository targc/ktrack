package ktrack

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type TrackItem struct {
	APIGroup  string
	Kind      string
	Namespace string
	Labels    map[string]string
}

type Resource struct {
	APIGroup  string
	Kind      string
	Namespace string
	Name      string
	Labels    map[string]string
	Status    map[string]interface{}
}

type Tracker struct {
	dynamic   dynamic.Interface
	discovery discovery.DiscoveryInterface
	interval  time.Duration
	items     []TrackItem

	mu       sync.Mutex
	gvrCache map[string]schema.GroupVersionResource
}

func New(cfg *rest.Config, interval time.Duration, items []TrackItem) (*Tracker, error) {
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &Tracker{
		dynamic:   dynClient,
		discovery: discClient,
		interval:  interval,
		items:     items,
		gvrCache:  make(map[string]schema.GroupVersionResource),
	}, nil
}

// Run blocks until ctx is cancelled, calling fn for every matched resource on every poll cycle.
func (t *Tracker) Run(ctx context.Context, fn func(Resource) error) error {
	for {
		t.poll(ctx, fn)
		timer := time.NewTimer(t.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (t *Tracker) poll(ctx context.Context, fn func(Resource) error) {
	for _, item := range t.items {
		gvr, err := t.resolveGVR(ctx, item.APIGroup, item.Kind)
		if err != nil {
			slog.Error(
				"resolve gvr error",
				"error", err.Error(),
				"api_group", item.APIGroup,
				"kind", item.Kind,
			)
			continue
		}

		ri := t.dynamic.Resource(gvr)
		var ri2 dynamic.ResourceInterface
		if item.Namespace != "" {
			ri2 = ri.Namespace(item.Namespace)
		} else {
			ri2 = ri
		}

		selector := labelMapToSelector(item.Labels)
		var continueToken string
		for {
			result, err := ri2.List(ctx, metav1.ListOptions{
				LabelSelector: selector,
				Limit:         500,
				Continue:      continueToken,
			})

			if err != nil {
				slog.Error("resource list error", "error", err.Error())
				break
			}

			if len(result.Items) == 0 {
				slog.Warn("resource list items is empty")
			}

			for _, obj := range result.Items {
				status, _ := obj.Object["status"].(map[string]interface{})
				fn(Resource{ //nolint:errcheck
					APIGroup:  item.APIGroup,
					Kind:      item.Kind,
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
					Labels:    obj.GetLabels(),
					Status:    status,
				})
			}

			if result.GetContinue() == "" {
				break
			}

			continueToken = result.GetContinue()
		}
	}
}

func (t *Tracker) resolveGVR(ctx context.Context, apiGroup, kind string) (schema.GroupVersionResource, error) {
	key := apiGroup + "/" + kind

	t.mu.Lock()
	if gvr, ok := t.gvrCache[key]; ok {
		t.mu.Unlock()
		return gvr, nil
	}
	t.mu.Unlock()

	groupResources, err := restmapper.GetAPIGroupResources(t.discovery)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to get API group resources: %w", err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := mapper.RESTMapping(schema.GroupKind{Group: apiGroup, Kind: kind})
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to map %s/%s: %w", apiGroup, kind, err)
	}

	t.mu.Lock()
	t.gvrCache[key] = mapping.Resource
	t.mu.Unlock()

	return mapping.Resource, nil
}

func labelMapToSelector(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
