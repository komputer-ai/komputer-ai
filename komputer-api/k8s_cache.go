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

// StartCache starts informers for cachedTypes and waits for their first sync.
// If they don't sync within timeout (e.g. the API's RBAC lacks watch), it stops
// them and list reads stay on the direct client.
func (k *K8sClient) StartCache(ctx context.Context, timeout time.Duration) {
	c, err := cache.New(k.restConfig, cache.Options{
		Scheme:           k.client.Scheme(),
		DefaultTransform: cache.TransformStripManagedFields(),
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
	k.cache = c
	Logger.Info("list cache synced")
}

// reader serves list reads from the cache once it has synced, else the direct client.
func (k *K8sClient) reader() client.Reader {
	if k.cache != nil {
		return k.cache
	}
	return k.client
}
