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
