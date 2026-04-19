# ktrack

Go library for polling Kubernetes resource status by apiGroup + kind + label selector.

## Architecture

- `tracker.go` — `Tracker` struct, `New()`, `Run()`, `poll()`, `resolveGVR()`, `labelMapToSelector()`
- `examples/basic/main.go` — usage example with in-cluster + kubeconfig fallback

## Key Design Decisions

- Uses `dynamic.Interface` + `discovery.DiscoveryInterface` — supports any resource including CRDs
- `APIGroup` not `apiVersion` — REST mapper resolves preferred version automatically; core resources use `""`
- GVR cached after first resolution per `apiGroup/kind` key; never expires (GVRs rarely change)
- Pagination: 500 items per page via `ListOptions.Limit` + `Continue` token
- `Resource.Status` is raw `map[string]interface{}` from `.status` — caller interprets it
- `Run` polls immediately then sleeps interval using `time.NewTimer` (not `time.After`, to avoid timer leak on ctx cancel)
- Items polled sequentially per cycle

## Public API

```go
func New(cfg *rest.Config, interval time.Duration, items []TrackItem) (*Tracker, error)
func (t *Tracker) Run(ctx context.Context, fn func(Resource) error) error
```
