package resource

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// Predicate defines a function filter out resources disabled for certain operations
type Predicate func(unstructured.Unstructured) bool

// AndPredicates combines multiple predicates into one, returning true only if all predicates return true
func AndPredicates(predicates ...Predicate) Predicate {
	return func(u unstructured.Unstructured) bool {
		for _, fn := range predicates {
			if !fn(u) {
				return false
			}
		}

		return true
	}
}

// OrPredicates combines multiple predicates into one, returning true if any predicate returns true
func OrPredicates(predicates ...Predicate) Predicate {
	return func(u unstructured.Unstructured) bool {
		for _, fn := range predicates {
			if fn(u) {
				return true
			}
		}
		return false
	}
}

// SplitByPredicates splits a list of unstructured objects into two lists
// one that matches all the given predicates and one that does not
func SplitByPredicates(objs []unstructured.Unstructured, predicates Predicate) ([]unstructured.Unstructured, []unstructured.Unstructured) {
	if predicates == nil {
		return nil, objs
	}

	matched := []unstructured.Unstructured{}
	notMatched := []unstructured.Unstructured{}
	for _, u := range objs {
		if predicates(u) {
			matched = append(matched, u)
		} else {
			notMatched = append(notMatched, u)
		}
	}
	return matched, notMatched
}

func IsCRD(u unstructured.Unstructured) bool {
	return HasKind("CustomResourceDefinition")(u)
}

func IsDeployment(u unstructured.Unstructured) bool {
	return HasKind("Deployment")(u)
}

func HasKind(kind string) Predicate {
	return func(u unstructured.Unstructured) bool {
		return u.GetKind() == kind
	}
}

func HasAnnotation(key, value string) Predicate {
	return func(u unstructured.Unstructured) bool {
		return hasElemInMap(u.GetAnnotations(), key, value)
	}
}

func HasLabel(key, value string) Predicate {
	return func(u unstructured.Unstructured) bool {
		return hasElemInMap(u.GetLabels(), key, value)
	}
}

func hasElemInMap(m map[string]string, key, value string) bool {
	if m == nil {
		return false
	}

	v, exists := m[key]
	return exists && v == value
}
