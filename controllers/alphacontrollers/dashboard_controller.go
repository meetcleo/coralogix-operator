/*
Copyright 2023.

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

package alphacontrollers

import (
	"context"
	"math/rand"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coralogix/coralogix-operator/controllers/clientset"
	dash "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/coralogix-dashboards/v1"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"

	"github.com/golang/protobuf/jsonpb"
	"github.com/nsf/jsondiff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var dashboardFinalizerName = "dashboard.coralogix.com/finalizer"

// DashboardReconciler reconciles a ConfigMap object
type DashboardReconciler struct {
	client.Client
	CoralogixClientSet clientset.ClientSetInterface
	Scheme             *runtime.Scheme
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

//+kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *DashboardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	jsm := &jsonpb.Marshaler{
		EmitDefaults: true,
	}
	dashClient := r.CoralogixClientSet.Dashboards()

	var dashboard *dash.Dashboard

	//Get ruleGroupSetRD
	configMap := &v1.ConfigMap{}
	if err := r.Client.Get(ctx, req.NamespacedName, configMap); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if value, ok := configMap.Labels["app.coralogix.com/dashboard"]; ok && value == "true" {
		id, idOk := configMap.Labels["app.coralogix.com/dashboard_id"]

		if !idOk {
			if configMap.Labels == nil {
				configMap.Labels = make(map[string]string)
			}
			if value, ok := configMap.Data["contentJson"]; ok {
				dashboard = new(dash.Dashboard)
				if err := protojson.Unmarshal([]byte(value), dashboard); err != nil {
					id = string(uuid.NewString())
					log.Error(err, "Error on unmarshalling dashboard")
				} else {
					id = dashboard.Id.GetValue()
					if id == "" {
						id = string(RandStringBytes(21))
					}
				}
			} else {
				id = string(RandStringBytes(21))
			}
			configMap.Labels["app.coralogix.com/dashboard_id"] = id

			if err := r.Update(ctx, configMap); err != nil {
				log.Error(err, "Error on updating ConfigMap Labels", "Name", configMap.Name, "Namespace", configMap.Namespace)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		}

		// examine DeletionTimestamp to determine if object is under deletion
		if configMap.ObjectMeta.DeletionTimestamp.IsZero() {
			// The object is not being deleted, so if it does not have our finalizer,
			// then lets add the finalizer and update the object. This is equivalent
			// registering our finalizer.
			if !controllerutil.ContainsFinalizer(configMap, dashboardFinalizerName) {
				controllerutil.AddFinalizer(configMap, dashboardFinalizerName)
				if err := r.Update(ctx, configMap); err != nil {
					log.Error(err, "Received an error while Updating a ConfigMap", "ConfigMap Name", configMap.Name)
					return ctrl.Result{}, err
				}
			}
		} else {
			// The object is being deleted
			if controllerutil.ContainsFinalizer(configMap, dashboardFinalizerName) {
				// our finalizer is present, so lets handle any external dependency
				if !idOk {
					controllerutil.RemoveFinalizer(configMap, dashboardFinalizerName)
					err := r.Update(ctx, configMap)
					log.Error(err, "Received an error while Updating a configMap", "configMap Name", configMap.Name)
					return ctrl.Result{}, err
				}

				deleteDashReq := &dash.DeleteDashboardRequest{DashboardId: &wrapperspb.StringValue{Value: id}}
				log.V(1).Info("Deleting Dashboard", "dashboard ID", id)
				if _, err := dashClient.DeleteDashboard(ctx, deleteDashReq); err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried unless it is deleted manually.
					log.Error(err, "Received an error while Deleting a Dashboard", "dashboard ID", id)
					if status.Code(err) == codes.NotFound {
						controllerutil.RemoveFinalizer(configMap, dashboardFinalizerName)
						err := r.Update(ctx, configMap)
						return ctrl.Result{}, err
					}
					return ctrl.Result{}, err
				}

				log.V(1).Info("Dashboard was deleted", "Dashboard ID", id)
				// remove our finalizer from the list and update it.
				controllerutil.RemoveFinalizer(configMap, dashboardFinalizerName)
				if err := r.Update(ctx, configMap); err != nil {
					return ctrl.Result{}, err
				}
			}

			// Stop reconciliation as the item is being deleted
			return ctrl.Result{}, nil
		}

		var notFound bool
		var err error
		var actualState []byte
		var expectedState []byte

		if id, ok := configMap.Labels["app.coralogix.com/dashboard_id"]; !ok {
			log.V(1).Info("Dashboard wasn't created in Coralogix backend")
			notFound = true
		} else if getDashboardResp, err := dashClient.GetDashboard(ctx, &dash.GetDashboardRequest{DashboardId: &wrapperspb.StringValue{Value: id}}); status.Code(err) == codes.NotFound {
			log.V(1).Info("Dashboard doesn't exist in Coralogix backend")
			notFound = true
		} else if err == nil {
			dashboard = getDashboardResp.Dashboard
			if actualState, err = protojson.Marshal(dashboard); err != nil {
				log.Error(err, "Received an error while Reading a Dashboard", "dashboard ID", id)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		}

		if value, ok := configMap.Data["contentJson"]; ok {
			dashboard = new(dash.Dashboard)
			if err := protojson.Unmarshal([]byte(value), dashboard); err != nil {
				log.Error(err, "Received an error while Creating a Dashboard", "dashboard ID", id)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
			dashboard.Id = &wrapperspb.StringValue{Value: id}
			expectedDashboard := new(dash.Dashboard)
			if err := protojson.Unmarshal([]byte(value), expectedDashboard); err != nil {
				log.Error(err, "Received an error while Creating a Dashboard", "dashboard ID", id)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
			expectedDashboard.Id = &wrapperspb.StringValue{Value: id}
			expectedDashboard.Folder = nil
			if expectedState, err = protojson.Marshal(expectedDashboard); err != nil {
				log.Error(err, "Received an error while Creating a Dashboard", "dashboard ID", id)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		}

		if notFound {
			createDashboardReq := &dash.CreateDashboardRequest{Dashboard: dashboard}
			jstr, _ := jsm.MarshalToString(createDashboardReq)
			log.V(1).Info("Creating Dashboard", "Dashboard", jstr)
			if createDashResp, err := dashClient.CreateDashboard(ctx, createDashboardReq); err == nil {
				jstr, _ := jsm.MarshalToString(createDashResp)
				log.V(1).Info("Dashboard was created", "Dashboard", jstr)
				if getDashboardResp, err := dashClient.GetDashboard(ctx, &dash.GetDashboardRequest{DashboardId: &wrapperspb.StringValue{Value: id}}); status.Code(err) == codes.NotFound {
					log.Error(err, "Received an error while getting a Dashboard", "Dashboard ID", id)
					return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
				} else {
					dashboard = getDashboardResp.Dashboard
					dashboard.Folder = nil
					if actualState, err = protojson.Marshal(dashboard); err != nil {
						log.Error(err, "Received an error while Reading a Dashboard", "dashboard ID", id)
						return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
					}

					if diffType, diffString := jsondiff.Compare(expectedState, actualState, &jsondiff.Options{}); !(diffType == jsondiff.FullMatch || diffType == jsondiff.SupersetMatch) {
						log.Error(err, "ContentJson does not match the dashboard content", "Diff", diffString)
						return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
					}
				}

				return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
			} else {
				log.Error(err, "Received an error while creating a Dashboard", "dashboard", createDashboardReq)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		} else if err != nil {
			log.Error(err, "Received an error while reading a Dashboard", "dashboard ID", id)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}

		if diffType, diffString := jsondiff.Compare(expectedState, actualState, &jsondiff.Options{}); !(diffType == jsondiff.FullMatch || diffType == jsondiff.SupersetMatch) {
			log.V(1).Info("Find diffs between spec and the actual state", "Diff", diffString)
			if value, ok := configMap.Data["contentJson"]; ok {
				dashboard = new(dash.Dashboard)
				if err := protojson.Unmarshal([]byte(value), dashboard); err != nil {
					log.Error(err, "Received an error while replacing a Dashboard", "dashboard ID", id)
					return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
				}
				dashboard.Id = &wrapperspb.StringValue{Value: id}
			}
			updateDashReq := &dash.ReplaceDashboardRequest{Dashboard: dashboard}
			if updateDashResp, err := dashClient.UpdateDashboard(ctx, updateDashReq); err != nil {
				log.Error(err, "Received an error while updating a Dashboard", "dashboard", updateDashReq)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			} else {
				jstr, _ := jsm.MarshalToString(updateDashResp)
				log.V(1).Info("Dashboard was updated on backend", "dashboard", jstr)
				if getDashboardResp, err := dashClient.GetDashboard(ctx, &dash.GetDashboardRequest{DashboardId: &wrapperspb.StringValue{Value: id}}); status.Code(err) == codes.NotFound {
					log.Error(err, "Received an error while getting a Dashboard", "Dashboard ID", id)
					return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
				} else {
					dashboard = getDashboardResp.Dashboard
					dashboard.Folder = nil
					if actualState, err = protojson.Marshal(dashboard); err != nil {
						log.Error(err, "Received an error while Reading a Dashboard", "dashboard ID", id)
						return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
					}

					if diffType, diffString := jsondiff.Compare(expectedState, actualState, &jsondiff.Options{}); !(diffType == jsondiff.FullMatch || diffType == jsondiff.SupersetMatch) {
						log.Error(err, "ContentJson does not match the dashboard content", "Diff", diffString)
						return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
					}
				}
			}
		}

		return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
	}
	return ctrl.Result{}, nil
}

func (r *DashboardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		Complete(r)
}
