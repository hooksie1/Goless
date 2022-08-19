/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	functionv1beta1 "github.com/hooksie1/goless/api/v1beta1"
)

var (
	FnUnavailable = "Unavailable"
	FnBuilding    = "Building"
	FnAvailable   = "Available"
	FnUpdate      = "Update"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=goless.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=goless.io,resources=functions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=goless.io,resources=functions/finalizers,verbs=update
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Function object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("function", req.NamespacedName)

	var function functionv1beta1.Function
	var existingDep appsv1.Deployment
	var existingCm corev1.ConfigMap
	var existingSvc corev1.Service

	if err := r.Get(ctx, req.NamespacedName, &function); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, err
	}

	if function.Status.Status == "" {
		function.Status.Status = FnUnavailable
	}

	// Configmap section

	cm := function.ConfigMap(req.Name, req.Namespace)
	if err := ctrl.SetControllerReference(&function, &cm, r.Scheme); err != nil {
		log.Error(err, "unable to set controller reference for configmap")
		return ctrl.Result{}, err
	}

	err := r.Get(ctx, req.NamespacedName, &existingCm)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "unable to get existing configmap")
		return ctrl.Result{}, err
	}

	if err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, &cm); err != nil {
			log.Error(err, "error creating configmap")
			return ctrl.Result{}, nil
		}
		r.Recorder.Event(&function, "Normal", "Created", "Configmap created")
	}

	if !equality.Semantic.DeepEqual(cm.Data, existingCm.Data) && !errors.IsNotFound(err) {
		function.Status.Status = FnUpdate
		if err := r.Update(ctx, &cm); err != nil {
			log.Error(err, "error updating configmap")
			return ctrl.Result{}, nil
		}
		r.Recorder.Event(&function, "Normal", "Updated", "Configmap updated")
		function.Status.Status = "Update"
	}

	// Service Section

	svc := function.Service(req.Name, req.Namespace)
	if err := ctrl.SetControllerReference(&function, &svc, r.Scheme); err != nil {
		log.Error(err, "unable to set controller reference for service")
		return ctrl.Result{}, err
	}

	err = r.Get(ctx, req.NamespacedName, &existingSvc)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "unable to get existing service")
		return ctrl.Result{}, err
	}

	if err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, &svc); err != nil {
			log.Error(err, "error creating service")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&function, "Normal", "Created", "Service created")
	}

	if !errors.IsNotFound(err) && function.GetServerPort32() != existingSvc.Spec.Ports[0].Port {
		svc.ResourceVersion = existingSvc.ResourceVersion
		if err := r.Update(ctx, &svc); err != nil {
			log.Error(err, "unable to update service port")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&function, "Normal", "Updated", "Service updated")
		function.Status.Status = "Update"
	}

	// Deployment section

	dep := function.Deployment(req.Name, req.Namespace)
	if err := ctrl.SetControllerReference(&function, &dep, r.Scheme); err != nil {
		log.Error(err, "unable to set controller reference for deployment")
		return ctrl.Result{}, err
	}

	err = r.Get(ctx, req.NamespacedName, &existingDep)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "unable to get existing deployment")
		return ctrl.Result{}, err
	}

	// Deployment is not found, so it's a new deployment
	if err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, &dep); err != nil {
			log.Error(err, "error creating deployment")
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&function, "Normal", "Created", "Deployment created")
	}

	if function.Status.Status == FnUpdate {
		fmt.Println("in update")
		now := time.Now()
		dep.Spec.Template.Annotations = map[string]string{
			"timestamp": now.String(),
		}
		if err := r.Update(ctx, &dep); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&function, "Normal", "Updated", "Deployment updated")
	}

	if !errors.IsNotFound(err) {
		if *dep.Spec.Replicas != *existingDep.Spec.Replicas || dep.Spec.Template.Spec.Containers[0].Env[0].Value != existingDep.Spec.Template.Spec.Containers[0].Env[0].Value {
			if err := r.Update(ctx, &dep); err != nil {
				log.Error(err, "unable to update existing deployment")
				return ctrl.Result{}, err
			}
			r.Recorder.Event(&function, "Normal", "Updated", "Deployment updated")
		}
	}

	for _, v := range existingDep.Status.Conditions {
		if v.Type == "Available" && v.Status != "True" {
			function.Status.Status = FnBuilding
			r.Client.Status().Update(ctx, &function)
			log.Info("function still initializing")
			return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
		}
	}

	function.Status.Status = FnAvailable
	r.Client.Status().Update(ctx, &function)
	log.Info("function successfully intiialized")
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&functionv1beta1.Function{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
