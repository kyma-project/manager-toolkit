package chart

import (
	"fmt"

	"github.com/kyma-project/manager-toolkit/installation/chart/annotation"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Install(config *Config, customFlags map[string]interface{}) error {
	return install(config, customFlags, renderChart)
}

func install(config *Config, customFlags map[string]interface{}, renderChartFunc func(config *Config, customFlags map[string]interface{}) (*release.Release, error)) error {
	cachedManifest, currentManifest, err := getCachedAndCurrentManifest(config, customFlags, renderChartFunc)
	if err != nil {
		return err
	}

	objs, unusedObjs, err := getObjectsToInstallAndRemove(cachedManifest, currentManifest)
	if err != nil {
		return err
	}

	err = updateObjects(config, objs)
	if err != nil {
		return err
	}

	err = uninstallObjects(config, unusedObjs)
	if err != nil {
		return err
	}

	return config.Cache.Set(config.Ctx, config.CacheKey, ContextManifest{
		ManagerUID:  config.ManagerUID,
		CustomFlags: customFlags,
		Manifest:    currentManifest,
	})
}

func getObjectsToInstallAndRemove(cachedManifest string, currentManifest string) ([]unstructured.Unstructured, []unstructured.Unstructured, error) {
	objs, err := parseManifest(currentManifest)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse chart manifest: %s", err.Error())
	}

	oldObjs, err := parseManifest(cachedManifest)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse chart manifest: %s", err.Error())
	}

	unusedObjs := unusedOldObjects(oldObjs, objs)
	return objs, unusedObjs, nil
}

func updateObjects(config *Config, objs []unstructured.Unstructured) error {
	for i := range objs {
		u := objs[i]
		config.Log.Debugf("creating %s %s/%s", u.GetKind(), u.GetNamespace(), u.GetName())

		u = annotation.AddDoNotEditDisclaimer(config.ManagerName, u)

		//TODO: implement pre-apply hook

		// TODO: what if Apply returns error in the middle of manifest?
		// maybe we should in this case translate applied objs into manifest and set it into cache?
		err := config.Cluster.Client.Apply(config.Ctx, client.ApplyConfigurationFromUnstructured(&u), &client.ApplyOptions{
			Force:        ptr.To(true),
			FieldManager: config.ManagerName,
		})
		if err != nil {
			return fmt.Errorf("could not install object %s/%s: %s", u.GetNamespace(), u.GetName(), err.Error())
		}
	}
	return nil
}

func unusedOldObjects(previousObjs []unstructured.Unstructured, currentObjs []unstructured.Unstructured) []unstructured.Unstructured {
	currentNames := make(map[string]struct{}, len(currentObjs))
	for _, obj := range currentObjs {
		objFullName := fmt.Sprintf("%s/%s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
		currentNames[objFullName] = struct{}{}
	}
	result := []unstructured.Unstructured{}
	for _, obj := range previousObjs {
		objFullName := fmt.Sprintf("%s/%s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
		if _, found := currentNames[objFullName]; !found {
			result = append(result, obj)
		}
	}
	return result
}
