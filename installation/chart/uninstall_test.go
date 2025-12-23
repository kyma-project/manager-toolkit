package chart

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/manager-toolkit/installation/base/resource"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_Uninstall(t *testing.T) {
	log := zap.NewNop().Sugar()

	testManifestKey := types.NamespacedName{
		Name: "test", Namespace: "testnamespace",
	}
	emptyManifestKey := types.NamespacedName{
		Name: "empty", Namespace: "manifest",
	}
	wrongManifestKey := types.NamespacedName{
		Name: "wrong", Namespace: "manifest",
	}

	cache := NewInMemoryManifestCache()
	_ = cache.Set(context.Background(), testManifestKey,
		ContextManifest{Manifest: fmt.Sprint(testCRD, separator, testDeploy)})
	_ = cache.Set(context.Background(), emptyManifestKey,
		ContextManifest{Manifest: ""})
	_ = cache.Set(context.Background(), wrongManifestKey,
		ContextManifest{Manifest: "api: test\n\tversion: test"})

	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}}

	type args struct {
		config *Config
		opts   UninstallOpts
	}
	tests := []struct {
		name            string
		args            args
		wantUninstalled bool
		wantErr         bool
	}{
		{
			name: "uninstall deploy as first",
			args: args{
				opts: UninstallOpts{
					UninstallFirst: resource.IsDeployment,
				},
				config: &Config{
					Ctx:      context.Background(),
					Log:      log,
					Cache:    cache,
					CacheKey: testManifestKey,
					Cluster: Cluster{
						Client: fake.NewClientBuilder().
							WithScheme(scheme.Scheme).
							WithObjects(testDeployCR.DeepCopy()).
							Build(),
					},
				},
			},
			wantUninstalled: false,
			wantErr:         false,
		},
		{
			name: "parse manifest error",
			args: args{
				config: &Config{
					Cache:    cache,
					CacheKey: wrongManifestKey,
				},
			},
			wantUninstalled: false,
			wantErr:         true,
		},
		{
			name: "uninstallation error",
			args: args{
				config: &Config{
					Ctx:      context.Background(),
					Log:      log,
					Cache:    cache,
					CacheKey: testManifestKey,
					Cluster: Cluster{
						Client: fake.NewClientBuilder().
							WithScheme(scheme.Scheme).
							WithObjects(ns.DeepCopy()).
							Build(),
					},
				},
			},
			wantUninstalled: true,
			wantErr:         false,
		},
		{
			name: "empty manifest",
			args: args{
				config: &Config{
					Cache:    cache,
					CacheKey: emptyManifestKey,
					Cluster: Cluster{
						Client: fake.NewClientBuilder().
							WithScheme(scheme.Scheme).
							WithObjects(ns.DeepCopy()).
							Build(),
					},
				},
			},
			wantUninstalled: true,
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uninstalled, err := Uninstall(tt.args.config, &tt.args.opts)
			require.Equal(t, tt.wantUninstalled, uninstalled)
			if (err != nil) != tt.wantErr {
				t.Errorf("uninstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
