// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"go.uber.org/zap/zapcore"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const managedByLabelKey = "app.kubernetes.io/managed-by"

const (
	defaultNotificationPeriodMinutes int64 = 5
	incidentIoIntegrationName              = "incident-io"
)

//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

//+kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get

// PrometheusRuleReconciler reconciles a PrometheusRule object
type PrometheusRuleReconciler struct {
	Interval time.Duration
}

func (r *PrometheusRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	prometheusRule := &prometheus.PrometheusRule{}
	if err := config.GetClient().Get(ctx, req.NamespacedName, prometheusRule); err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{}, err
	}

	var errs error
	if shouldTrackRecordingRules(prometheusRule) {
		err := r.convertPrometheusRuleRecordingRuleToCxRecordingRule(ctx, log, prometheusRule, req)
		if err != nil {
			log.Error(err, "Received an error while trying to convert PrometheusRule to RecordingRule CRD")
			errs = errors.Join(errs, err)
		}
	} else {
		err := r.deleteCxRecordingRule(ctx, req)
		if err != nil {
			log.Error(err, "Received an error while trying to delete RecordingRule CRD")
			errs = errors.Join(errs, err)
		}
	}

	if shouldTrackAlerts(prometheusRule) {
		err := r.convertPrometheusRuleAlertToCxAlert(ctx, prometheusRule)
		if err != nil {
			log.Error(err, "Received an error while trying to convert PrometheusRule to Alert CRD")
			errs = errors.Join(errs, err)
		}
	} else {
		err := r.deleteCxAlerts(ctx, prometheusRule)
		if err != nil {
			log.Error(err, "Received an error while trying to delete Alert CRDs")
			errs = errors.Join(errs, err)
		}
	}

	if errs != nil {
		return ctrl.Result{}, errs
	}

	return reconcile.Result{RequeueAfter: r.Interval}, nil
}

func (r *PrometheusRuleReconciler) convertPrometheusRuleRecordingRuleToCxRecordingRule(ctx context.Context, log logr.Logger, prometheusRule *prometheus.PrometheusRule, req reconcile.Request) error {
	desiredRecordingRuleGroupSetSpec := prometheusRuleToRecordingRuleToRuleGroupSet(log, prometheusRule)
	if len(desiredRecordingRuleGroupSetSpec.Groups) == 0 {
		return nil
	}

	recordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{}
	if err := config.GetClient().Get(ctx, req.NamespacedName, recordingRuleGroupSet); err != nil {
		if k8serrors.IsNotFound(err) {
			recordingRuleGroupSet.Name = prometheusRule.Name
			recordingRuleGroupSet.Namespace = prometheusRule.Namespace
			recordingRuleGroupSet.Labels = prometheusRule.Labels
			recordingRuleGroupSet.Labels[managedByLabelKey] = truncateLabelValue(prometheusRule.Name)
			recordingRuleGroupSet.OwnerReferences = []metav1.OwnerReference{getOwnerReference(prometheusRule)}
			recordingRuleGroupSet.Spec = desiredRecordingRuleGroupSetSpec
			if err = config.GetClient().Create(ctx, recordingRuleGroupSet); err != nil {
				return fmt.Errorf("received an error while trying to create RecordingRuleGroupSet CRD: %w", err)
			}
			return nil
		}

		return fmt.Errorf("received an error while trying to get RecordingRuleGroupSet CRD: %w", err)
	}

	updated := false
	desiredLabels := prometheusRule.Labels
	desiredLabels[managedByLabelKey] = truncateLabelValue(prometheusRule.Name)
	if !reflect.DeepEqual(recordingRuleGroupSet.Labels, desiredLabels) {
		recordingRuleGroupSet.Labels = desiredLabels
		updated = true
	}

	desiredOwnerReferences := []metav1.OwnerReference{getOwnerReference(prometheusRule)}
	if !reflect.DeepEqual(recordingRuleGroupSet.OwnerReferences, desiredOwnerReferences) {
		recordingRuleGroupSet.OwnerReferences = desiredOwnerReferences
		updated = true
	}

	if !reflect.DeepEqual(recordingRuleGroupSet.Spec, desiredRecordingRuleGroupSetSpec) {
		recordingRuleGroupSet.Spec = desiredRecordingRuleGroupSetSpec
		updated = true
	}

	if updated {
		if err := config.GetClient().Update(ctx, recordingRuleGroupSet); err != nil {
			return fmt.Errorf("received an error while trying to update RecordingRuleGroupSet CRD: %w", err)
		}
	}

	return nil
}

func (r *PrometheusRuleReconciler) deleteCxRecordingRule(ctx context.Context, req reconcile.Request) error {
	recordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}

	if err := config.GetClient().Delete(ctx, recordingRuleGroupSet); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("received an error while trying to delete RecordingRuleGroupSet CRD: %w", err)
	}

	return nil
}

func (r *PrometheusRuleReconciler) convertPrometheusRuleAlertToCxAlert(ctx context.Context, prometheusRule *prometheus.PrometheusRule) error {
	// A single PrometheusRule can have multiple alerts with the same name, while the Alert CRD from coralogix can only manage one alert.
	// alertMap is used to map an alert name with potentially multiple alerts from the promrule CRD. For example:
	//
	// A prometheusRule with the following rules:
	// rules:
	//   - alert: Example
	//     expr: metric > 10
	//   - alert: Example
	//     expr: metric > 20
	//
	// Would be mapped into:
	//   map[string][]prometheus.Rule{
	// 	   "Example": []prometheus.Rule{
	// 		 {
	//          Alert: Example,
	//          Expr: "metric > 10"
	// 		 },
	// 		 {
	//          Alert: Example,
	//          Expr: "metric > 100"
	// 		 },
	// 	   },
	//   }
	//
	// To later on generate coralogix Alert CRDs using the alert name followed by it's index on the array, making sure we don't clash names.
	alertMap := make(map[string][]prometheus.Rule)
	var a string
	for _, group := range prometheusRule.Spec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert != "" {
				a = strings.ToLower(rule.Alert)
				if _, ok := alertMap[a]; !ok {
					alertMap[a] = []prometheus.Rule{rule}
					continue
				}
				alertMap[a] = append(alertMap[a], rule)
			}
		}
	}

	alertsToKeep := make(map[string]bool)
	var errorsEncountered []error
	for alertName, rules := range alertMap {
		for i, rule := range rules {
			alert := &coralogixv1beta1.Alert{}
			alertName := fmt.Sprintf("%s-%s-%d",
				prometheusRule.Name,
				sanitizeName(alertName),
				i,
			)
			alertsToKeep[alertName] = true
			if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: prometheusRule.Namespace, Name: alertName}, alert); err != nil {
				if k8serrors.IsNotFound(err) {
					alert.Name = alertName
					alert.Namespace = prometheusRule.Namespace
					alert.Labels = prometheusRule.Labels
					alert.Labels[managedByLabelKey] = truncateLabelValue(prometheusRule.Name)
					alert.OwnerReferences = []metav1.OwnerReference{getOwnerReference(prometheusRule)}
					alert.Spec = prometheusAlertingRuleToAlertSpec(&rule)
					if err = config.GetClient().Create(ctx, alert); err != nil {
						errorsEncountered = append(errorsEncountered, fmt.Errorf("error creating Alert CRD %s: %w", alertName, err))
					}
					continue
				}
				errorsEncountered = append(errorsEncountered, fmt.Errorf("error getting Alert CRD %s: %w", alertName, err))
				continue
			}

			updated := false
			desiredLabels := prometheusRule.Labels
			desiredLabels[managedByLabelKey] = truncateLabelValue(prometheusRule.Name)
			if !reflect.DeepEqual(alert.Labels, desiredLabels) {
				alert.Labels = desiredLabels
				updated = true
			}

			desiredOwnerReferences := []metav1.OwnerReference{getOwnerReference(prometheusRule)}
			if !reflect.DeepEqual(alert.OwnerReferences, desiredOwnerReferences) {
				alert.OwnerReferences = desiredOwnerReferences
				updated = true
			}

			desiredDescription := rule.Annotations["description"]
			if alert.Spec.Description != desiredDescription {
				alert.Spec.Description = desiredDescription
				updated = true
			}

			desiredEntityLabels := rule.Labels
			if !reflect.DeepEqual(alert.Spec.EntityLabels, desiredEntityLabels) {
				alert.Spec.EntityLabels = desiredEntityLabels
				updated = true
			}

			desiredPriority := getPriority(rule)
			if alert.Spec.Priority != desiredPriority {
				alert.Spec.Priority = desiredPriority
				updated = true
			}

			// Only reconcile the notification group when the PrometheusRule defines the
			// annotations; otherwise preserve a manually-tuned NotificationGroup on the
			// Alert CR (see #295 - don't overwrite advanced Alert fields).
			if desiredNotificationGroup := prometheusAlertToNotificationGroup(rule); desiredNotificationGroup != nil {
				if !reflect.DeepEqual(alert.Spec.NotificationGroup, desiredNotificationGroup) {
					alert.Spec.NotificationGroup = desiredNotificationGroup
					updated = true
				}
			}

			desiredTypeDefinition := coralogixv1beta1.AlertTypeDefinition{
				MetricThreshold: prometheusAlertToMetricThreshold(rule, desiredPriority),
			}
			// Preserve a manually-tuned MinNonNullValuesPct on the Alert CR (see #295),
			// unless the PrometheusRule explicitly sets it via annotation.
			if _, ok := rule.Annotations["cxMinNonNullValuesPct"]; !ok {
				desiredTypeDefinition.MetricThreshold.MissingValues.MinNonNullValuesPct = alert.Spec.TypeDefinition.MetricThreshold.MissingValues.MinNonNullValuesPct
			}
			if !reflect.DeepEqual(alert.Spec.TypeDefinition, desiredTypeDefinition) {
				alert.Spec.TypeDefinition = desiredTypeDefinition
				updated = true
			}

			if updated {
				if err := config.GetClient().Update(ctx, alert); err != nil {
					errorsEncountered = append(errorsEncountered, fmt.Errorf("error updating Alert CRD %s: %w", alertName, err))
				}
			}
		}
	}

	var childAlerts coralogixv1beta1.AlertList
	if err := config.GetClient().List(
		ctx,
		&childAlerts,
		client.InNamespace(prometheusRule.Namespace),
		client.MatchingLabels{managedByLabelKey: truncateLabelValue(prometheusRule.Name)}); err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	for _, alert := range childAlerts.Items {
		if !alertsToKeep[alert.Name] {
			if err := config.GetClient().Delete(ctx, &alert); err != nil {
				errorsEncountered = append(errorsEncountered, fmt.Errorf("error deleting Alert CRD %s: %w", alert.Name, err))
			}
		}
	}

	if len(errorsEncountered) > 0 {
		return errors.Join(errorsEncountered...)
	}
	return nil
}

func (r *PrometheusRuleReconciler) deleteCxAlerts(ctx context.Context, prometheusRule *prometheus.PrometheusRule) error {
	var childAlerts coralogixv1beta1.AlertList
	err := config.GetClient().List(
		ctx,
		&childAlerts,
		client.InNamespace(prometheusRule.Namespace),
		client.MatchingLabels{managedByLabelKey: truncateLabelValue(prometheusRule.Name)})
	if err != nil {
		return fmt.Errorf("received an error while trying to list Alerts: %w", err)
	}

	var errs error
	for _, alert := range childAlerts.Items {
		if err := config.GetClient().Delete(ctx, &alert); err != nil {
			if k8serrors.IsNotFound(err) {
				return nil
			}
			errs = errors.Join(errs, fmt.Errorf("received an error while trying to delete Alert CRD %s: %w", alert.Name, err))
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}

func shouldTrackRecordingRules(prometheusRule *prometheus.PrometheusRule) bool {
	if value, ok := prometheusRule.Labels[utils.TrackPrometheusRuleRecordingRulesLabelKey]; ok && value == "true" {
		return true
	}
	if value, ok := prometheusRule.Labels[utils.KubernetesComponentLabelKey]; ok && value == utils.KubernetesComponentSLO {
		return true
	}
	return false
}

func shouldTrackAlerts(prometheusRule *prometheus.PrometheusRule) bool {
	if value, ok := prometheusRule.Labels[utils.TrackPrometheusRuleAlertsLabelKey]; ok && value == "true" {
		return true
	}
	if value, ok := prometheusRule.Labels[utils.KubernetesComponentLabelKey]; ok && value == utils.KubernetesComponentSLO {
		return true
	}
	return false
}

func prometheusAlertingRuleToAlertSpec(rule *prometheus.Rule) coralogixv1beta1.AlertSpec {
	priority := getPriority(*rule)
	return coralogixv1beta1.AlertSpec{
		Name:              rule.Alert,
		Description:       rule.Annotations["description"],
		EntityLabels:      rule.Labels,
		Priority:          priority,
		NotificationGroup: prometheusAlertToNotificationGroup(*rule),
		TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
			MetricThreshold: prometheusAlertToMetricThreshold(*rule, priority),
		},
	}
}

// prometheusAlertToNotificationGroup builds a NotificationGroup from annotations:
// cxIntegrationName sets a custom outbound-webhook integration, notifyToIncidentIo
// (on/true) adds the incident-io integration, and cxNotifyEveryMin overrides the
// re-trigger period (defaulting to the rule's `for` duration, then to 5 minutes).
func prometheusAlertToNotificationGroup(rule prometheus.Rule) *coralogixv1beta1.NotificationGroup {
	notificationPeriod := defaultNotificationPeriodMinutes
	if cxNotifyEveryMin, ok := rule.Annotations["cxNotifyEveryMin"]; ok {
		if minutes, err := strconv.Atoi(cxNotifyEveryMin); err == nil {
			notificationPeriod = int64(minutes)
		}
	} else if rule.For != nil {
		if duration, err := time.ParseDuration(string(*rule.For)); err == nil {
			notificationPeriod = int64(duration.Minutes())
		}
	}
	if notificationPeriod == 0 {
		notificationPeriod = defaultNotificationPeriodMinutes
	}

	var integrationNames []string
	if integrationName, ok := rule.Annotations["cxIntegrationName"]; ok {
		integrationNames = append(integrationNames, integrationName)
	}
	if notify, ok := rule.Annotations["notifyToIncidentIo"]; ok && (notify == "on" || notify == "true") {
		integrationNames = append(integrationNames, incidentIoIntegrationName)
	}

	if len(integrationNames) == 0 {
		return nil
	}

	webhooks := make([]coralogixv1beta1.WebhookSettings, len(integrationNames))
	for i, name := range integrationNames {
		webhooks[i] = coralogixv1beta1.WebhookSettings{
			RetriggeringPeriod: coralogixv1beta1.RetriggeringPeriod{
				Minutes: ptr.To(notificationPeriod),
			},
			NotifyOn: coralogixv1beta1.NotifyOnTriggeredAndResolved,
			Integration: coralogixv1beta1.IntegrationType{
				IntegrationRef: &coralogixv1beta1.IntegrationRef{
					BackendRef: &coralogixv1beta1.OutboundWebhookBackendRef{
						Name: ptr.To(name),
					},
				},
			},
		}
	}

	return &coralogixv1beta1.NotificationGroup{
		Webhooks: webhooks,
	}
}

func prometheusRuleToRecordingRuleToRuleGroupSet(log logr.Logger, prometheusRule *prometheus.PrometheusRule) coralogixv1alpha1.RecordingRuleGroupSetSpec {
	groups := make([]coralogixv1alpha1.RecordingRuleGroup, 0)
	for _, group := range prometheusRule.Spec.Groups {
		var interval int32 = 60

		if group.Interval != nil && *group.Interval != "" {
			duration, err := time.ParseDuration(string(*group.Interval))
			if err != nil {
				log.V(int(zapcore.WarnLevel)).Info("Failed to parse interval duration", "interval", group.Interval, "error", err, "using default interval")
			}

			// Convert duration to seconds
			durationSeconds := int32(duration.Seconds())

			if durationSeconds < interval {
				log.V(int(zapcore.WarnLevel)).Info("Recording rule interval is lower than the default interval", "interval", durationSeconds, "default interval", interval, "using the greater interval")
			} else {
				// Update interval if parsed duration is greater
				interval = durationSeconds
			}
		}

		if rules := prometheusInnerRulesToCoralogixInnerRules(group.Rules); len(rules) > 0 {
			groups = append(groups, coralogixv1alpha1.RecordingRuleGroup{
				Name:            group.Name,
				IntervalSeconds: interval,
				Rules:           rules,
			})
		}
	}

	return coralogixv1alpha1.RecordingRuleGroupSetSpec{
		Groups: groups,
	}
}

func prometheusInnerRulesToCoralogixInnerRules(rules []prometheus.Rule) []coralogixv1alpha1.RecordingRule {
	result := make([]coralogixv1alpha1.RecordingRule, 0)
	for _, rule := range rules {
		if rule.Record == "" {
			continue
		}

		result = append(result, coralogixv1alpha1.RecordingRule{
			Record: rule.Record,
			Expr:   rule.Expr.StrVal,
			Labels: rule.Labels,
		})
	}
	return result
}

func prometheusAlertToMetricThreshold(rule prometheus.Rule, priority coralogixv1beta1.AlertPriority) *coralogixv1beta1.MetricThreshold {
	return &coralogixv1beta1.MetricThreshold{
		MetricFilter: coralogixv1beta1.MetricFilter{
			Promql: rule.Expr.StrVal,
		},
		Rules: []coralogixv1beta1.MetricThresholdRule{
			{
				Condition: coralogixv1beta1.MetricThresholdRuleCondition{
					Threshold:  resource.MustParse("0"),
					ForOverPct: 100,
					OfTheLast: coralogixv1beta1.MetricTimeWindow{
						DynamicDuration: ptr.To(string(ptr.Deref(rule.For, "1m"))),
					},
					ConditionType: coralogixv1beta1.MetricThresholdConditionTypeMoreThan,
				},
				Override: &coralogixv1beta1.AlertOverride{
					Priority: priority,
				},
			},
		},
		MissingValues: prometheusAlertToMissingValues(rule),
	}
}

// prometheusAlertToMissingValues maps the cxReplaceWithZero and cxMinNonNullValuesPct
// annotations onto the alert's missing-values handling, defaulting to 0% when unset.
func prometheusAlertToMissingValues(rule prometheus.Rule) coralogixv1beta1.MetricMissingValues {
	missingValues := coralogixv1beta1.MetricMissingValues{
		MinNonNullValuesPct: ptr.To(int64(0)),
	}
	if cxReplaceWithZero, ok := rule.Annotations["cxReplaceWithZero"]; ok {
		if replaceWithZero, err := strconv.ParseBool(cxReplaceWithZero); err == nil {
			missingValues.ReplaceWithZero = replaceWithZero
		}
	}
	if cxMinNonNullValuesPct, ok := rule.Annotations["cxMinNonNullValuesPct"]; ok {
		if pct, err := strconv.Atoi(cxMinNonNullValuesPct); err == nil {
			missingValues.MinNonNullValuesPct = ptr.To(int64(pct))
		}
	}
	return missingValues
}

func getPriority(rule prometheus.Rule) coralogixv1beta1.AlertPriority {
	if severityStr, ok := rule.Labels["severity"]; ok && severityStr != "" {
		if priority, ok := prometheusSeverityToCXPriority[strings.ToLower(severityStr)]; ok {
			return priority
		}
	}
	// Unset, empty, "none", or unrecognized severities are treated as P5/no escalation.
	return coralogixv1beta1.AlertPriorityP5
}

var prometheusSeverityToCXPriority = map[string]coralogixv1beta1.AlertPriority{
	"critical": coralogixv1beta1.AlertPriorityP1,
	"error":    coralogixv1beta1.AlertPriorityP2,
	"warning":  coralogixv1beta1.AlertPriorityP3,
	"info":     coralogixv1beta1.AlertPriorityP4,
	"low":      coralogixv1beta1.AlertPriorityP5,
}

func getOwnerReference(promRule *prometheus.PrometheusRule) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: promRule.APIVersion,
		Kind:       promRule.Kind,
		Name:       promRule.Name,
		UID:        promRule.UID,
	}
}

// sanitizeName converts any string into a valid Kubernetes resource name
// according to RFC 1123 (lowercase alphanumeric, '-', or '.').
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	// Replace any invalid characters (anything not a-z, 0-9, '-', or '.') with '-'
	re := regexp.MustCompile(`[^a-z0-9.-]+`)
	name = re.ReplaceAllString(name, "-")
	// Trim leading/trailing non-alphanumeric characters
	name = strings.Trim(name, "-.")
	return name
}

// truncateLabelValue ensures label values stay under 63 chars.
func truncateLabelValue(value string) string {
	if len(value) <= 63 {
		return value
	}
	h := sha1.Sum([]byte(value))
	return fmt.Sprintf("%s-%s", value[:40], hex.EncodeToString(h[:])[:8])
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	shouldTrackPrometheusRules := func(labels map[string]string) bool {
		if value, ok := labels[utils.TrackPrometheusRuleRecordingRulesLabelKey]; ok && value == "true" {
			return true
		}
		if value, ok := labels[utils.TrackPrometheusRuleAlertsLabelKey]; ok && value == "true" {
			return true
		}
		if value, ok := labels[utils.KubernetesComponentLabelKey]; ok && value == utils.KubernetesComponentSLO {
			return true
		}
		return false
	}

	selector := config.GetConfig().Selector
	return ctrl.NewControllerManagedBy(mgr).
		For(&prometheus.PrometheusRule{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return shouldTrackPrometheusRules(e.Object.GetLabels()) &&
					selector.Matches(e.Object.GetLabels(), e.Object.GetNamespace())
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return (shouldTrackPrometheusRules(e.ObjectNew.GetLabels()) ||
					shouldTrackPrometheusRules(e.ObjectOld.GetLabels())) &&
					(selector.Matches(e.ObjectNew.GetLabels(), e.ObjectNew.GetNamespace()) ||
						selector.Matches(e.ObjectOld.GetLabels(), e.ObjectOld.GetNamespace()))
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return shouldTrackPrometheusRules(e.Object.GetLabels()) &&
					selector.Matches(e.Object.GetLabels(), e.Object.GetNamespace())
			},
		}).
		Complete(r)
}
