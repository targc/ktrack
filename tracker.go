package ktrack

import "time"

type TrackItem struct {
	APIGroup string
	Kind     string
	Labels   map[string]string
}

type Resource struct {
	APIGroup string
	Kind     string
	Labels   map[string]string
	Name     string
	Status   string
}

type Tracker struct {
	intervalDuration time.Duration
	items            []TrackItem
}

func (t *Tracker) OnStatus(r Resource) error {
	return nil
}
