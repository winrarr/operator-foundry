package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConditionsReportKstatusLifecycle(t *testing.T) {
	var conditions []metav1.Condition
	markReconciling(&conditions, 3, "DependencyNotReady", "waiting for connection")
	assertConditionStatus(t, conditions, ConditionReady, metav1.ConditionFalse)
	assertConditionStatus(t, conditions, ConditionReconciling, metav1.ConditionTrue)
	assertConditionStatus(t, conditions, ConditionStalled, metav1.ConditionFalse)

	markReady(&conditions, 3, "resource is ready")
	assertConditionStatus(t, conditions, ConditionReady, metav1.ConditionTrue)
	assertConditionStatus(t, conditions, ConditionReconciling, metav1.ConditionFalse)
	assertConditionStatus(t, conditions, ConditionStalled, metav1.ConditionFalse)

	markStalled(&conditions, 4, "InvalidSpec", nil)
	assertConditionStatus(t, conditions, ConditionReady, metav1.ConditionFalse)
	assertConditionStatus(t, conditions, ConditionReconciling, metav1.ConditionFalse)
	assertConditionStatus(t, conditions, ConditionStalled, metav1.ConditionTrue)
	for _, condition := range conditions {
		if condition.ObservedGeneration != 4 {
			t.Fatalf("condition %s observed generation %d, want 4", condition.Type, condition.ObservedGeneration)
		}
	}
}

func assertConditionStatus(t *testing.T, conditions []metav1.Condition, conditionType string, expected metav1.ConditionStatus) {
	t.Helper()
	for _, condition := range conditions {
		if condition.Type == conditionType {
			if condition.Status != expected {
				t.Fatalf("condition %s status %s, want %s", conditionType, condition.Status, expected)
			}
			return
		}
	}
	t.Fatalf("condition %s not found in %#v", conditionType, conditions)
}
