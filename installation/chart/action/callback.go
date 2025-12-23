package action

import (
	"github.com/kyma-project/manager-toolkit/installation/base/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type PreApply func(*unstructured.Unstructured) error

// PreApplyWithPredicate wraps a PreApply function with predicates to filter which resources the callback is applied to
func PreApplyWithPredicate(applyFunc PreApply, predicate resource.Predicate) PreApply {
	return func(u *unstructured.Unstructured) error {
		if predicate(*u) {
			return applyFunc(u)
		}
		return nil
	}
}

func FireAllPreApply(actions []PreApply, u *unstructured.Unstructured) error {
	for _, f := range actions {
		if err := f(u); err != nil {
			return err
		}
	}
	return nil
}

type PostUninstall func(u unstructured.Unstructured) (bool, error)

// PostUninstallWithPredicate wraps a PostUninstall function with predicates to filter which resources the callback is applied to
func PostUninstallWithPredicate(postUninstallFunc PostUninstall, predicate resource.Predicate) PostUninstall {
	return func(u unstructured.Unstructured) (bool, error) {
		if predicate(u) {
			return postUninstallFunc(u)
		}
		return false, nil
	}
}

func FireAllPostUninstall(actions []PostUninstall, u unstructured.Unstructured) (bool, error) {
	done := true
	for _, f := range actions {
		d, err := f(u)
		if err != nil {
			return false, err
		}
		if !d {
			done = false
		}
	}
	return done, nil
}
