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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	testv1alpha1 "github.com/ncsibra/operator/api/v1alpha1"
)

// SensitiveReconciler reconciles a Sensitive object
type SensitiveReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=test.origoss.com,resources=sensitives,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.origoss.com,resources=sensitives/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.origoss.com,resources=sensitives/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sensitive object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SensitiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sensitive testv1alpha1.Sensitive
	if err := r.Get(ctx, req.NamespacedName, &sensitive); err != nil {
		if errors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("'%s' Sensitive deleted", req.NamespacedName))

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	var secret v1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: sensitive.Name, Namespace: sensitive.Namespace}, &secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensitive.Name,
				Namespace: sensitive.Namespace,
			},
			Data: map[string][]byte{
				sensitive.Spec.Key: []byte(sensitive.Spec.Value),
			},
		}

		err := controllerutil.SetControllerReference(&sensitive, secret, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, secret)
		if err != nil {
			logger.Error(err, "unable to create secret")
			return ctrl.Result{}, err
		}
	} else {
		owned := false
		for _, owner := range secret.OwnerReferences {
			if owner.UID == sensitive.UID {
				owned = true
				break
			}
		}

		if !owned {
			return ctrl.Result{}, errors.NewConflict(apiextensions.Resource(secret.Kind), secret.Name, fmt.Errorf("secret with name '%s' already exists, but not owned by Sensitive: '%s'", secret.Name, sensitive.Name))
		}

		oldSecretValue, ok := secret.Data[sensitive.Spec.Key]
		if !ok {
			return ctrl.Result{}, errors.NewConflict(apiextensions.Resource(secret.Kind), secret.Name, fmt.Errorf("unable to find secret value by key '%s'", sensitive.Spec.Key))
		}

		if string(oldSecretValue) == sensitive.Spec.Value {
			return ctrl.Result{}, nil
		}

		secret.Data[sensitive.Spec.Key] = []byte(sensitive.Spec.Value)

		err := r.Update(ctx, &secret)
		if err != nil {
			logger.Error(err, "unable to update secret")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SensitiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1alpha1.Sensitive{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
