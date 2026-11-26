package main

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// cachedTypes are the CRs whose list reads are served from watched, in-memory
// copies instead of the API server. A dashboard load lists agents several times
// over; without this, each one is a full LIST against the API server.
//
// Only list helpers and template lookups read through the cache. Writes and
// single-object Gets stay on the direct client, so read-modify-write paths
// (status patches, cost accumulation) never act on a stale copy.
var cachedTypes = []client.Object{
	&komputerv1alpha1.KomputerAgent{},
	&komputerv1alpha1.KomputerSquad{},
	&komputerv1alpha1.KomputerSchedule{},
	&komputerv1alpha1.KomputerOffice{},
	&komputerv1alpha1.KomputerSkill{},
	&komputerv1alpha1.KomputerMemory{},
	&komputerv1alpha1.KomputerConnector{},
	&komputerv1alpha1.KomputerAgentTemplate{},
	&komputerv1alpha1.KomputerAgentClusterTemplate{},
}

// listCache wraps the synced cache so it can be swapped in atomically.
type listCache struct{ client.Reader }

// StartCache starts informers for cachedTypes and waits for their first sync,
// then switches list reads over to them. Until then, and for good if they don't
// sync within timeout (e.g. the API's RBAC lacks watch), list reads go to the
// API server. main runs it in the background so startup never waits on it.
func (k *K8sClient) StartCache(ctx context.Context, timeout time.Duration) {
	c, err := cache.New(k.restConfig, cache.Options{
		Scheme:           k.client.Scheme(),
		DefaultTransform: cache.TransformStripManagedFields(),
		ByObject: map[client.Object]cache.ByObject{
			&komputerv1alpha1.KomputerAgent{}: {Transform: stripAgentHeavyFields},
			&komputerv1alpha1.KomputerSquad{}: {Transform: stripSquadHeavyFields},
		},
		// Reading any type not registered below is a bug, not a reason to
		// start a cluster-wide informer for it.
		ReaderFailOnMissingInformer: true,
	})
	if err != nil {
		Logger.Warnw("list cache disabled: create failed", "error", err)
		return
	}
	for _, obj := range cachedTypes {
		if _, err := c.GetInformer(ctx, obj); err != nil {
			Logger.Warnw("list cache disabled: register informer failed", "type", obj, "error", err)
			return
		}
	}

	cacheCtx, stop := context.WithCancel(ctx)
	go func() {
		if err := c.Start(cacheCtx); err != nil {
			Logger.Warnw("list cache stopped", "error", err)
		}
	}()
	syncCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if !c.WaitForCacheSync(syncCtx) {
		stop()
		Logger.Warnw("list cache disabled: informers did not sync; check the API's RBAC has watch on komputer resources", "timeout", timeout)
		return
	}
	// Synced: the informers run until ctx ends, so stop is only needed above.
	_ = stop
	k.cache.Store(&listCache{c})
	Logger.Info("list cache synced")
}

// stripAgentHeavyFields drops the agent fields that are usually most of an
// agent's size and that list reads served from the cache don't use: prompts,
// instructions and pod spec. Reads that need them use ListAgents, which skips
// the cache.
func stripAgentHeavyFields(obj any) (any, error) {
	if a, ok := obj.(*komputerv1alpha1.KomputerAgent); ok {
		a.ManagedFields = nil
		stripAgentSpec(&a.Spec)
	}
	return obj, nil
}

// stripSquadHeavyFields does the same for squad members' inline specs. Squad
// reads that need them (create, patch, add member) use GetSquad, which skips
// the cache.
func stripSquadHeavyFields(obj any) (any, error) {
	if s, ok := obj.(*komputerv1alpha1.KomputerSquad); ok {
		s.ManagedFields = nil
		for i := range s.Spec.Members {
			if s.Spec.Members[i].Spec != nil {
				stripAgentSpec(s.Spec.Members[i].Spec)
			}
		}
	}
	return obj, nil
}

func stripAgentSpec(spec *komputerv1alpha1.KomputerAgentSpec) {
	spec.Instructions = ""
	spec.SystemPrompt = ""
	spec.InternalSystemPrompt = ""
	spec.PodSpec = nil
}

// reader serves list reads from the cache once it has synced, else the direct client.
func (k *K8sClient) reader() client.Reader {
	if c := k.cache.Load(); c != nil {
		return c.Reader
	}
	return k.client
}
