// untested sections: 3

package matchers

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/matchers/internal/miter"
	"github.com/onsi/gomega/matchers/support/goraph/bipartitegraph"
)

type ConsistOfMatcher struct {
	Elements        []any
	missingElements []any
	extraElements   []any
}

func (matcher *ConsistOfMatcher) Match(actual any) (success bool, err error) {
	if !isArrayOrSlice(actual) && !isMap(actual) && !miter.IsIter(actual) {
		return false, fmt.Errorf("ConsistOf matcher expects an array/slice/map/iter.Seq/iter.Seq2.  Got:\n%s", format.Object(actual, 1))
	}

	matchers := matchers(matcher.Elements)
	values := valuesOf(actual)

	bipartiteGraph, err := bipartitegraph.NewBipartiteGraph(values, matchers, neighbours)
	if err != nil {
		return false, err
	}

	edges := bipartiteGraph.LargestMatching()
	if len(edges) == len(values) && len(edges) == len(matchers) {
		return true, nil
	}

	var missingMatchers []any
	matcher.extraElements, missingMatchers = bipartiteGraph.FreeLeftRight(edges)
	matcher.missingElements = equalMatchersToElements(missingMatchers)

	return false, nil
}

func neighbours(value, matcher any) (bool, error) {
	match, err := matcher.(omegaMatcher).Match(value)
	return match && err == nil, nil
}

func equalMatchersToElements(matchers []any) (elements []any) {
	for _, matcher := range matchers {
		if equalMatcher, ok := matcher.(*EqualMatcher); ok {
			elements = append(elements, equalMatcher.Expected)
		} else if _, ok := matcher.(*BeNilMatcher); ok {
			elements = append(elements, nil)
		} else {
			elements = append(elements, matcher)
		}
	}
	return
}

func flatten(elems []any) []any {
	if len(elems) != 1 ||
		!(isArrayOrSlice(elems[0]) ||
			(miter.IsIter(elems[0]) && !miter.IsSeq2(elems[0]))) {
		return elems
	}

	if miter.IsIter(elems[0]) {
		flattened := []any{}
		miter.IterateV(elems[0], func(v reflect.Value) bool {
			flattened = append(flattened, v.Interface())
			return true
		})
		return flattened
	}

	value := reflect.ValueOf(elems[0])
	flattened := make([]any, value.Len())
	for i := 0; i < value.Len(); i++ {
		flattened[i] = value.Index(i).Interface()
	}
	return flattened
}

func matchers(expectedElems []any) (matchers []any) {
	for _, e := range flatten(expectedElems) {
		if e == nil {
			matchers = append(matchers, &BeNilMatcher{})
		} else if matcher, isMatcher := e.(omegaMatcher); isMatcher {
			matchers = append(matchers, matcher)
		} else {
			matchers = append(matchers, &EqualMatcher{Expected: e})
		}
	}
	return
}

func presentable(elems []any) any {
	elems = flatten(elems)

	if len(elems) == 0 {
		return []any{}
	}

	sv := reflect.ValueOf(elems)
	firstEl := sv.Index(0)
	if firstEl.IsNil() {
		return elems
	}
	tt := firstEl.Elem().Type()
	for i := 1; i < sv.Len(); i++ {
		el := sv.Index(i)
		if el.IsNil() || (sv.Index(i).Elem().Type() != tt) {
			return elems
		}
	}

	ss := reflect.MakeSlice(reflect.SliceOf(tt), sv.Len(), sv.Len())
	for i := 0; i < sv.Len(); i++ {
		ss.Index(i).Set(sv.Index(i).Elem())
	}

	return ss.Interface()
}

func valuesOf(actual any) []any {
	value := reflect.ValueOf(actual)
	values := []any{}
	if miter.IsIter(actual) {
		if miter.IsSeq2(actual) {
			miter.IterateKV(actual, func(k, v reflect.Value) bool {
				values = append(values, v.Interface())
				return true
			})
		} else {
			miter.IterateV(actual, func(v reflect.Value) bool {
				values = append(values, v.Interface())
				return true
			})
		}
	} else if isMap(actual) {
		keys := value.MapKeys()
		for i := 0; i < value.Len(); i++ {
			values = append(values, value.MapIndex(keys[i]).Interface())
		}
	} else {
		for i := 0; i < value.Len(); i++ {
			values = append(values, value.Index(i).Interface())
		}
	}

	return values
}

func (matcher *ConsistOfMatcher) FailureMessage(actual any) (message string) {
	message = format.Message(actual, "to consist of", presentable(matcher.Elements))
	message = appendMissingElements(message, matcher.missingElements)
	if len(matcher.extraElements) > 0 {
		message = fmt.Sprintf("%s\nthe extra elements were\n%s", message,
			format.Object(presentable(matcher.extraElements), 1))
	}
	return
}

func appendMissingElements(message string, missingElements []any) string {
	if len(missingElements) == 0 {
		return message
	}
	return fmt.Sprintf("%s\nthe missing elements were\n%s", message,
		format.Object(presentable(missingElements), 1))
}

func (matcher *ConsistOfMatcher) NegatedFailureMessage(actual any) (message string) {
	return format.Message(actual, "not to consist of", presentable(matcher.Elements))
}
