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
	"testing"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

func TestShouldTrackRules(t *testing.T) {
	tests := []struct {
		name            string
		labels          map[string]string
		expectRecording bool
		expectAlerting  bool
	}{
		{name: "no labels", labels: nil, expectRecording: false, expectAlerting: false},
		{name: "track recording label", labels: map[string]string{utils.TrackPrometheusRuleRecordingRulesLabelKey: "true"}, expectRecording: true, expectAlerting: false},
		{name: "track alerting label", labels: map[string]string{utils.TrackPrometheusRuleAlertsLabelKey: "true"}, expectRecording: false, expectAlerting: true},
		{name: "SLO component tracks both", labels: map[string]string{utils.KubernetesComponentLabelKey: utils.KubernetesComponentSLO}, expectRecording: true, expectAlerting: true},
		{name: "non-SLO component tracks neither", labels: map[string]string{utils.KubernetesComponentLabelKey: "controller"}, expectRecording: false, expectAlerting: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &prometheus.PrometheusRule{ObjectMeta: metav1.ObjectMeta{Labels: tt.labels}}
			assert.Equal(t, tt.expectRecording, shouldTrackRecordingRules(pr))
			assert.Equal(t, tt.expectAlerting, shouldTrackAlerts(pr))
		})
	}
}

func TestPrometheusAlertToMissingValues(t *testing.T) {
	tests := []struct {
		name            string
		annotations     map[string]string
		expectReplace   bool
		expectMinNonNil int64
	}{
		{name: "defaults", annotations: nil, expectReplace: false, expectMinNonNil: 0},
		{name: "replace with zero", annotations: map[string]string{"cxReplaceWithZero": "true"}, expectReplace: true, expectMinNonNil: 0},
		{name: "min non-null pct", annotations: map[string]string{"cxMinNonNullValuesPct": "80"}, expectReplace: false, expectMinNonNil: 80},
		{name: "both", annotations: map[string]string{"cxReplaceWithZero": "true", "cxMinNonNullValuesPct": "60"}, expectReplace: true, expectMinNonNil: 60},
		{name: "invalid values ignored", annotations: map[string]string{"cxReplaceWithZero": "nope", "cxMinNonNullValuesPct": "abc"}, expectReplace: false, expectMinNonNil: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mv := prometheusAlertToMissingValues(prometheus.Rule{Annotations: tt.annotations})
			assert.Equal(t, tt.expectReplace, mv.ReplaceWithZero)
			assert.Equal(t, ptr.To(tt.expectMinNonNil), mv.MinNonNullValuesPct)
		})
	}
}

func TestPrometheusAlertToNotificationGroup(t *testing.T) {
	t.Run("no annotations yields no group", func(t *testing.T) {
		assert.Nil(t, prometheusAlertToNotificationGroup(prometheus.Rule{}))
	})

	t.Run("custom integration name", func(t *testing.T) {
		ng := prometheusAlertToNotificationGroup(prometheus.Rule{
			Annotations: map[string]string{"cxIntegrationName": "my-webhook"},
		})
		assert.Len(t, ng.Webhooks, 1)
		assert.Equal(t, "my-webhook", *ng.Webhooks[0].Integration.IntegrationRef.BackendRef.Name)
		assert.Equal(t, coralogixv1beta1.NotifyOnTriggeredAndResolved, ng.Webhooks[0].NotifyOn)
		assert.Equal(t, ptr.To(defaultNotificationPeriodMinutes), ng.Webhooks[0].RetriggeringPeriod.Minutes)
	})

	t.Run("notify to incident io", func(t *testing.T) {
		for _, v := range []string{"on", "true"} {
			ng := prometheusAlertToNotificationGroup(prometheus.Rule{
				Annotations: map[string]string{"notifyToIncidentIo": v},
			})
			assert.Len(t, ng.Webhooks, 1)
			assert.Equal(t, incidentIoIntegrationName, *ng.Webhooks[0].Integration.IntegrationRef.BackendRef.Name)
		}
	})

	t.Run("both integrations", func(t *testing.T) {
		ng := prometheusAlertToNotificationGroup(prometheus.Rule{
			Annotations: map[string]string{"cxIntegrationName": "my-webhook", "notifyToIncidentIo": "on"},
		})
		assert.Len(t, ng.Webhooks, 2)
		assert.Equal(t, "my-webhook", *ng.Webhooks[0].Integration.IntegrationRef.BackendRef.Name)
		assert.Equal(t, incidentIoIntegrationName, *ng.Webhooks[1].Integration.IntegrationRef.BackendRef.Name)
	})

	t.Run("cxNotifyEveryMin overrides period", func(t *testing.T) {
		ng := prometheusAlertToNotificationGroup(prometheus.Rule{
			Annotations: map[string]string{"cxIntegrationName": "my-webhook", "cxNotifyEveryMin": "30"},
		})
		assert.Equal(t, ptr.To(int64(30)), ng.Webhooks[0].RetriggeringPeriod.Minutes)
	})

	t.Run("period derived from for duration", func(t *testing.T) {
		ng := prometheusAlertToNotificationGroup(prometheus.Rule{
			Annotations: map[string]string{"cxIntegrationName": "my-webhook"},
			For:         ptr.To(prometheus.Duration("15m")),
		})
		assert.Equal(t, ptr.To(int64(15)), ng.Webhooks[0].RetriggeringPeriod.Minutes)
	})
}
