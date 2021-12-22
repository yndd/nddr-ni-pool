/*
Copyright 2021 NDD.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nipool

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/ndd-runtime/pkg/meta"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nipoolv1alpha1 "github.com/yndd/nddr-ni-pool/apis/nipool/v1alpha1"
	"github.com/yndd/nddr-ni-pool/internal/pool"
	"github.com/yndd/nddr-ni-pool/internal/shared"
)

const (
	finalizerName = "finalizer.alloc.nipool.nddr.yndd.io"
	//
	reconcileTimeout = 1 * time.Minute
	longWait         = 1 * time.Minute
	mediumWait       = 30 * time.Second
	shortWait        = 15 * time.Second
	veryShortWait    = 5 * time.Second

	// Errors
	errGetK8sResource = "cannot get nipool resource"
	errUpdateStatus   = "cannot update status of nipool resource"

	// events
	reasonReconcileSuccess      event.Reason = "ReconcileSuccess"
	reasonCannotDelete          event.Reason = "CannotDeleteResource"
	reasonCannotAddFInalizer    event.Reason = "CannotAddFinalizer"
	reasonCannotDeleteFInalizer event.Reason = "CannotDeleteFinalizer"
	reasonCannotInitialize      event.Reason = "CannotInitializeResource"
	reasonCannotGetAllocations  event.Reason = "CannotGetAllocations"
	reasonAppLogicFailed        event.Reason = "ApplogicFailed"
	reasonCannotGarbageCollect  event.Reason = "CannotGarbageCollect"
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// Reconciler reconciles packages.
type Reconciler struct {
	client  resource.ClientApplicator
	log     logging.Logger
	record  event.Recorder
	managed mrManaged

	newNiPool func() nipoolv1alpha1.Np
	pool      map[string]pool.Pool
}

type mrManaged struct {
	resource.Finalizer
}

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

func WithNewNiPoolFn(f func() nipoolv1alpha1.Np) ReconcilerOption {
	return func(r *Reconciler) {
		r.newNiPool = f
	}
}

func WithPool(t map[string]pool.Pool) ReconcilerOption {
	return func(r *Reconciler) {
		r.pool = t
	}
}

// WithRecorder specifies how the Reconciler should record Kubernetes events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

func defaultMRManaged(m ctrl.Manager) mrManaged {
	return mrManaged{
		Finalizer: resource.NewAPIFinalizer(m.GetClient(), finalizerName),
	}
}

// Setup adds a controller that reconciles nipools.
func Setup(mgr ctrl.Manager, o controller.Options, nddcopts *shared.NddControllerOptions) error {
	name := "nddr/" + strings.ToLower(nipoolv1alpha1.NiPoolGroupKind)
	ap := func() nipoolv1alpha1.Np { return &nipoolv1alpha1.NiPool{} }

	r := NewReconciler(mgr,
		WithLogger(nddcopts.Logger.WithValues("controller", name)),
		WithNewNiPoolFn(ap),
		WithPool(nddcopts.Pool),
		WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	allocHandler := &EnqueueRequestForAllAllocations{
		client: mgr.GetClient(),
		log:    nddcopts.Logger,
		ctx:    context.Background(),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&nipoolv1alpha1.NiPool{}).
		Watches(&source.Kind{Type: &nipoolv1alpha1.Alloc{}}, allocHandler).
		WithEventFilter(resource.IgnoreUpdateWithoutGenerationChangePredicate()).
		Complete(r)
}

// NewReconciler creates a new reconciler.
func NewReconciler(mgr ctrl.Manager, opts ...ReconcilerOption) *Reconciler {

	r := &Reconciler{
		client: resource.ClientApplicator{
			Client:     mgr.GetClient(),
			Applicator: resource.NewAPIPatchingApplicator(mgr.GetClient()),
		},
		log:     logging.NewNopLogger(),
		record:  event.NewNopRecorder(),
		managed: defaultMRManaged(mgr),
		pool:    make(map[string]pool.Pool),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

// Reconcile ipam allocation.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) { // nolint:gocyclo
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling nipool", "NameSpace", req.NamespacedName)

	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	cr := r.newNiPool()
	if err := r.client.Get(ctx, req.NamespacedName, cr); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		log.Debug("Cannot get managed resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetK8sResource)
	}
	record := r.record.WithAnnotations("name", cr.GetAnnotations()[cr.GetName()])

	treename := strings.Join([]string{cr.GetNamespace(), cr.GetName()}, "/")

	// initialize the status
	if meta.WasDeleted(cr) {
		log = log.WithValues("deletion-timestamp", cr.GetDeletionTimestamp())

		if cr.GetAllocations() != 0 {
			record.Event(cr, event.Normal(reasonCannotDelete, "allocations present"))
			log.Debug("Allocations present we cannot delete the pool")

			if _, ok := r.pool[treename]; !ok {
				r.log.Debug("pool/tree init", "treename", treename)
				r.pool[treename] = pool.New(cr.GetSize(), cr.GetAllocationStrategy())
			}

			if err := r.GarbageCollection(ctx, cr, treename); err != nil {
				record.Event(cr, event.Warning(reasonCannotGarbageCollect, err))
				log.Debug("Cannot perform garbage collection", "error", err)
				cr.SetConditions(nddv1.ReconcileError(err), nipoolv1alpha1.NotReady())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
			}

			return reconcile.Result{RequeueAfter: mediumWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}

		if err := r.managed.RemoveFinalizer(ctx, cr); err != nil {
			// If this is the first time we encounter this issue we'll be
			// requeued implicitly when we update our status with the new error
			// condition. If not, we requeue explicitly, which will trigger
			// backoff.
			record.Event(cr, event.Warning(reasonCannotDeleteFInalizer, err))
			log.Debug("Cannot remove managed resource finalizer", "error", err)
			cr.SetConditions(nddv1.ReconcileError(err), nipoolv1alpha1.NotReady())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
		}

		delete(r.pool, req.NamespacedName.String())

		// We've successfully deallocated our external resource (if necessary) and
		// removed our finalizer. If we assume we were the only controller that
		// added a finalizer to this resource then it should no longer exist and
		// thus there is no point trying to update its status.
		log.Debug("Successfully deallocated resource")
		return reconcile.Result{Requeue: false}, nil
	}

	if err := r.managed.AddFinalizer(ctx, cr); err != nil {
		// If this is the first time we encounter this issue we'll be requeued
		// implicitly when we update our status with the new error condition. If
		// not, we requeue explicitly, which will trigger backoff.
		record.Event(cr, event.Warning(reasonCannotAddFInalizer, err))
		log.Debug("Cannot add finalizer", "error", err)
		cr.SetConditions(nddv1.ReconcileError(err), nipoolv1alpha1.NotReady())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	// initialize resource
	if err := cr.InitializeResource(); err != nil {
		record.Event(cr, event.Warning(reasonCannotInitialize, err))
		log.Debug("Cannot initialize", "error", err)
		cr.SetConditions(nddv1.ReconcileError(err), nipoolv1alpha1.NotReady())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	if err := r.handleAppLogic(ctx, cr, treename); err != nil {
		record.Event(cr, event.Warning(reasonAppLogicFailed, err))
		log.Debug("handle applogic failed", "error", err)
		cr.SetConditions(nddv1.ReconcileError(err), nipoolv1alpha1.NotReady())
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	if err := r.GarbageCollection(ctx, cr, treename); err != nil {
		record.Event(cr, event.Warning(reasonCannotGarbageCollect, err))
		log.Debug("Cannot perform garbage collection", "error", err)
		cr.SetConditions(nddv1.ReconcileError(err), nipoolv1alpha1.NotReady())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
	}

	cr.SetConditions(nddv1.ReconcileSuccess(), nipoolv1alpha1.Ready())
	return reconcile.Result{RequeueAfter: reconcileTimeout}, errors.Wrap(r.client.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *Reconciler) handleAppLogic(ctx context.Context, cr nipoolv1alpha1.Np, treename string) error {
	// initialize the pool
	if _, ok := r.pool[treename]; !ok {
		r.log.Debug("pool/tree init", "treename", treename)
		r.pool[treename] = pool.New(cr.GetSize(), cr.GetAllocationStrategy())
	}

	return nil
}

func (r *Reconciler) GarbageCollection(ctx context.Context, cr nipoolv1alpha1.Np, treename string) error {
	log := r.log.WithValues("function", "garbageCollection", "Name", cr.GetName())
	//log.Debug("entry")
	// get all allocations
	alloc := &nipoolv1alpha1.AllocList{}
	if err := r.client.List(ctx, alloc); err != nil {
		log.Debug("Cannot get allocations", "error", err)
		return err
	}

	log.Debug("Alloc List", "Alloc", alloc.Items)

	poolname := cr.GetName()

	log.Debug("PoolName", "poolname", poolname)

	// check if allocations dont have NI allocated
	// check if allocations match with allocated pool
	// -> alloc found in pool -> ok
	// -> alloc not found in pool -> assign in pool
	// we keep track of the allocated Nis to compare in a second stage
	allocNis := make([]*string, 0)
	for _, alloc := range alloc.Items {
		// only garbage collect if the pool matches
		if alloc.GetNiPoolName() == cr.GetName() {
			allocNi, allocNiFound := alloc.HasNi()
			if !allocNiFound {
				log.Debug("Alloc", "Pool", poolname, "Name", alloc.GetName(), "NI found", allocNiFound)
			} else {
				log.Debug("Alloc", "Pool", poolname, "Name", alloc.GetName(), "NI found", allocNiFound, "allocNI", allocNi)
			}

			// the selector is used in the pool to find the entry in the pool
			// we use the annotations with src-tag in the key
			// TBD how do we handle annotation
			selector := labels.NewSelector()
			sourcetags := make(map[string]string)
			for key, val := range alloc.GetSourceTag() {
				req, err := labels.NewRequirement(key, selection.In, []string{val})
				if err != nil {
					log.Debug("wrong object", "error", err)
				}
				selector = selector.Add(*req)
				sourcetags[key] = val
			}
			var ok bool

			var ni string
			var nis []string
			// query the pool to see if an allocation was performed using the selector
			switch cr.GetAllocationStrategy() {
			default:
				// first available allocation strategy
				nis = r.pool[treename].QueryByLabels(selector)
			}

			if len(nis) == 0 {
				// label/selector not found in the pool -> allocate Ni in pool
				// this can happen if the allocation was not assigned an Ni
				if allocNiFound {
					// can happen during startup that an allocation had already an NI
					// we need to ensure we allocate the same NI again
					switch cr.GetAllocationStrategy() {
					default:
						ni, ok = r.pool[treename].Allocate(allocNi, sourcetags)
					}
					log.Debug("strange situation: query not found in pool, but alloc has an NI (can happen during startup/restart)", "allocNI", allocNi, "new NI", ni)
				} else {
					switch cr.GetAllocationStrategy() {
					default:
						ni, ok = r.pool[treename].Allocate("", sourcetags)
					}
				}
				if !ok {
					log.Debug("pool allocation failed")
					return errors.New("Pool allocation failed")
				}
				// find Ni in the used list in the nipool resource
				if cr.FindNi(ni) {
					log.Debug("strange situation, query not found in pool, but AS was found in used allocations (can happen during startup/restart) ")
				}

			} else {
				// label/selector found in the pool
				ni = nis[0]
				if len(nis) > 1 {
					// this should never happen since the metalabels will point to the same entry
					// in the pool
					log.Debug("strange situation, NI found in pool multiple times", "ases", nis)
				}
				switch {
				case !allocNiFound:
					log.Debug("strange situation, NI found in pool but alloc NI not found")
				case allocNiFound && ni != allocNi:
					log.Debug("strange situation, NI found in pool but alloc NI had different NI", "pool NI", ni, "alloc NI", allocNi)
				default:
					// do nothing, all ok
				}
			}
			allocNis = append(allocNis, &ni)
		}
	}
	// based on the allocated ASes we collected before we can validate if the
	// pool had assigned other allocations
	found := false
	for _, pNi := range r.pool[treename].GetAllocated() {
		for _, allocNi := range allocNis {
			if *allocNi == pNi {
				found = true
				break
			}
		}
		if !found {
			log.Debug("NI found in pool, but no alloc found -> deallocate from pool", "as", pNi)
			r.pool[treename].DeAllocate(pNi)
		}
	}
	// always update the status field in the nipool with the latest info
	cr.UpdateNi(allocNis)

	return nil
}
