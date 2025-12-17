package chart

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_flagsBuilder_Build(t *testing.T) {
	t.Run("build empty flags", func(t *testing.T) {
		flags, err := NewFlagsBuilder().Build()
		require.NoError(t, err)
		require.Equal(t, map[string]interface{}{}, flags)
	})

	t.Run("build flags", func(t *testing.T) {
		expectedFlags := map[string]interface{}{
			"controllerManager": map[string]interface{}{
				"container": map[string]interface{}{
					"env": map[string]interface{}{
						"ISTIO_INSTALLED": "\"true\"",
					},
				},
			},
			"global": map[string]interface{}{
				"commonLabels": map[string]interface{}{
					"managedBy": "test-runner",
				},
				"images": map[string]interface{}{
					"connection": "conn-im",
					"rp":         "rp-im",
				},
			},
		}

		flags, err := NewFlagsBuilder().
			With("global.commonLabels.managedBy", "test-runner").
			With("global.images.connection", "conn-im").
			With("controllerManager.container.env.ISTIO_INSTALLED", "\"true\"").
			With("global.images.rp", "rp-im").Build()

		require.NoError(t, err)
		require.Equal(t, expectedFlags, flags)
	})

}
