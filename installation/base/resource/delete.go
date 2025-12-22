package resource

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete tries to delete the given unstructured object from the cluster
// It returns true if the object is already deleted or not found, false if the deletion is still in progress,
// and an error if any other error occurs during deletion
func Delete(ctx context.Context, c client.Client, log *zap.SugaredLogger, u unstructured.Unstructured) (bool, error) {
	log.Debugf("deleting %s %s", u.GetKind(), u.GetName())
	err := c.Delete(ctx, &u)
	if k8serrors.IsNotFound(err) {
		log.Debugf("deletion skipped for %s %s", u.GetKind(), u.GetName())
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not uninstall object %s/%s: %s", u.GetNamespace(), u.GetName(), err.Error())
	}

	// deletion in progress
	return false, nil
}
