package resource

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// NewRSAvailableReason is added in a deployment when its newest replica set is made available
	// ie. the number of new pods that have passed readiness checks and run for at least minReadySeconds
	// is at least the minimum available pods that need to run for the deployment.
	NewRSAvailableReason = "NewReplicaSetAvailable"

	// MinimumReplicasAvailableReason is added in a deployment when it has its minimum replicas required available.
	MinimumReplicasAvailableReason = "MinimumReplicasAvailable"
)

func IsDeploymentReady(deployment appsv1.Deployment) bool {
	conditions := deployment.Status.Conditions
	return HasDeploymentConditionTrueStatusWithReason(conditions, appsv1.DeploymentAvailable, MinimumReplicasAvailableReason) &&
		HasDeploymentConditionTrueStatusWithReason(conditions, appsv1.DeploymentProgressing, NewRSAvailableReason) &&
		deployment.Generation == deployment.Status.ObservedGeneration && // spec changes are observed
		deployment.Status.UnavailableReplicas == 0 // all replicas are available
}

func HasDeploymentConditionTrueStatus(conditions []appsv1.DeploymentCondition, conditionType appsv1.DeploymentConditionType) bool {
	condition := GetDeploymentCondition(conditions, conditionType)
	return condition.Status == corev1.ConditionTrue
}

func HasDeploymentConditionTrueStatusWithReason(conditions []appsv1.DeploymentCondition, conditionType appsv1.DeploymentConditionType, reason string) bool {
	condition := GetDeploymentCondition(conditions, conditionType)
	return condition.Status == corev1.ConditionTrue && condition.Reason == reason
}

func GetDeploymentCondition(conditions []appsv1.DeploymentCondition, conditionType appsv1.DeploymentConditionType) appsv1.DeploymentCondition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition
		}
	}
	return appsv1.DeploymentCondition{}
}
