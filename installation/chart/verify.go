package chart

import (
	"fmt"

	"github.com/kyma-project/manager-toolkit/installation/base/resource"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// VerificationCompleted indicates that the verification has been completed
	VerificationCompleted = "OK"

	// DeploymentVerificationProcessing indicates that the deployment is still being processed
	DeploymentVerificationProcessing = "DeploymentProcessing"
)

type VerificationResult struct {
	// Ready indicates whether the verification was successful
	Ready bool
	// Reason provides additional information about the verification result
	// It can be "OK" if the verification is complete, DeploymentProcessing if still in progress,
	// or a specific reason for failure.
	Reason string
}

// Verify checks the status of the deployed chart resources to determine if they are ready.
// If an error occurs during the verification process, it returns an error.
// It returns a VerificationResult indicating readiness and any relevant reason.
func Verify(config *Config) (*VerificationResult, error) {
	spec, err := config.Cache.Get(config.Ctx, config.CacheKey)
	if err != nil {
		return nil, fmt.Errorf("could not render manifest from chart: %s", err.Error())
	}
	// sometimes cache is not created yet
	if len(spec.Manifest) == 0 {
		return &VerificationResult{Ready: false}, nil
	}

	objs, err := parseManifest(spec.Manifest)
	if err != nil {
		return nil, fmt.Errorf("could not parse chart manifest: %s", err.Error())
	}

	for i := range objs {
		u := objs[i]

		if !resource.IsDeployment(u) {
			continue
		}

		reason, err := verifyDeployment(config, u)
		if err != nil {
			return nil, fmt.Errorf("could not verify deployment %s/%s: %s", u.GetNamespace(), u.GetName(), err.Error())
		}

		if reason != VerificationCompleted {
			return &VerificationResult{Ready: false, Reason: reason}, nil
		}
	}

	return &VerificationResult{Ready: true, Reason: VerificationCompleted}, nil
}

func verifyDeployment(config *Config, u unstructured.Unstructured) (string, error) {
	var deployment appsv1.Deployment
	err := config.Cluster.Client.Get(config.Ctx, types.NamespacedName{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
	}, &deployment)
	if err != nil {
		return "", err
	}

	if resource.IsDeploymentReady(deployment) {
		return VerificationCompleted, nil
	}

	if resource.HasDeploymentConditionTrueStatus(deployment.Status.Conditions, appsv1.DeploymentReplicaFailure) {
		return fmt.Sprintf("deployment %s/%s has replica failure: %s", u.GetNamespace(), u.GetName(),
			resource.GetDeploymentCondition(deployment.Status.Conditions, appsv1.DeploymentReplicaFailure).Message), nil
	}

	return DeploymentVerificationProcessing, nil
}
