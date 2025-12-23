package chart

import (
	"fmt"

	"github.com/kyma-project/manager-toolkit/installation/base/resource"
	"github.com/kyma-project/manager-toolkit/installation/chart/action"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type UninstallOpts struct {
	// Predicate can be used to uninstall certain resources before others
	// other resources will be uninstalled after these are done
	UninstallFirst resource.Predicate

	// PostActions to be executed after uninstalling each resource
	// can be used for cleanup tasks
	PostActions []action.PostUninstall
}

// Uninstall uninstalls all resources defined in the chart manifest stored in the cache
func Uninstall(config *Config, opts *UninstallOpts) (bool, error) {
	done, err := uninstall(config, opts)
	if err != nil {
		return done, err
	}

	if !done {
		// not all resources are deleted yet
		return done, nil
	}

	// all resources are deleted, remove the cache entry
	return true, config.Cache.Delete(config.Ctx, config.CacheKey)
}

func uninstall(config *Config, opts *UninstallOpts) (bool, error) {
	spec, err := config.Cache.Get(config.Ctx, config.CacheKey)
	if err != nil {
		return false, fmt.Errorf("could not render manifest from chart: %s", err.Error())
	}

	manifestObjs, err := parseManifest(spec.Manifest)
	if err != nil {
		return false, fmt.Errorf("could not parse chart manifest: %s", err.Error())
	}

	firstToUninstall, objs := resource.SplitByPredicates(manifestObjs, opts.UninstallFirst)

	// delete first to uninstall objs
	done, err := deleteObjects(config, firstToUninstall)
	if err != nil || !done {
		return done, err
	}

	// delete remaining objs
	done, err = deleteObjects(config, objs)
	if err != nil || !done {
		return done, err
	}

	// fire post uninstall actions for all objs
	return firePostUninstallForObjs(opts, manifestObjs)
}

func deleteObjects(config *Config, objs []unstructured.Unstructured) (bool, error) {
	done := true
	for i := range objs {
		u := objs[i]

		objDone, err := resource.Delete(config.Ctx, config.Cluster.Client, config.Log, u)
		if err != nil {
			return false, err
		}

		if !objDone {
			done = false
		}
	}

	return done, nil
}

func firePostUninstallForObjs(opts *UninstallOpts, objs []unstructured.Unstructured) (bool, error) {
	done := true
	for i := range objs {
		u := objs[i]

		objDone, err := action.FireAllPostUninstall(opts.PostActions, u)
		if err != nil {
			return false, err
		}

		if !objDone {
			done = false
		}
	}

	return done, nil
}
