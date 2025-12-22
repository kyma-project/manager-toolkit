package annotation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAddDoNotEditDisclaimer(t *testing.T) {
	t.Run("add disclaimer", func(t *testing.T) {
		obj := unstructured.Unstructured{}
		obj = AddDoNotEditDisclaimer("reconciler", obj)

		expectedAnnotation := fmt.Sprintf(annotationFormat, "reconciler", "reconciler")
		expectedMessage := fmt.Sprintf(messageFormat, "reconciler")

		require.Equal(t, expectedMessage, obj.GetAnnotations()[expectedAnnotation])
	})
}
