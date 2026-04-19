# ktrack

Go library for polling Kubernetes resource status. Configure which resources to watch by `apiGroup`, `kind`, and label selectors — get the raw `.status` object on every poll cycle.

## Install

```bash
go get github.com/targc/ktrack
```

## Usage

```go
cfg, _ := rest.InClusterConfig() // or clientcmd.BuildConfigFromFlags

tracker, err := ktrack.New(cfg, 30*time.Second, []ktrack.TrackItem{
    {APIGroup: "apps", Kind: "Deployment", Namespace: "production"},
    {APIGroup: "",     Kind: "Pod",        Labels: map[string]string{"app": "api"}},
    {APIGroup: "batch", Kind: "Job",       Namespace: "jobs"},
})

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

tracker.Run(ctx, func(r ktrack.Resource) error {
    fmt.Println(r.Kind, r.Namespace+"/"+r.Name, r.Status["phase"])
    return nil
})
```

## Types

```go
type TrackItem struct {
    APIGroup  string            // e.g. "apps", "batch", "" for core
    Kind      string            // e.g. "Deployment", "Pod"
    Namespace string            // empty = all namespaces
    Labels    map[string]string // optional label selector
}

type Resource struct {
    APIGroup  string
    Kind      string
    Namespace string
    Name      string
    Labels    map[string]string
    Status    map[string]interface{} // raw .status from k8s object
}
```

## Notes

- Uses `APIGroup` (not `apiVersion`) — the REST mapper resolves the preferred version automatically.
- GVR resolution is cached after the first poll; unknown kinds are silently skipped.
- List calls are paginated at 500 items per page.
- `Run` polls immediately on start, then waits the interval.
