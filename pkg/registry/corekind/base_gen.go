// Code generated - EDITING IS FUTILE. DO NOT EDIT.
//
// Generated by:
//     kinds/gen.go
// Using jennies:
//     BaseCoreRegistryJenny
//
// Run 'make gen-cue' from repository root to regenerate.

package corekind

import (
	"fmt"

	"github.com/grafana/grafana/pkg/kinds/test"
	"github.com/grafana/grafana/pkg/kindsys"
	"github.com/grafana/thema"
)

// Base is a registry of kindsys.Interface. It provides two modes for accessing
// kinds: individually via literal named methods, or as a slice returned from
// an All*() method.
//
// Prefer the individual named methods for use cases where the particular kind(s) that
// are needed are known to the caller. For example, a dashboard linter can know that it
// specifically wants the dashboard kind.
//
// Prefer All*() methods when performing operations generically across all kinds.
// For example, a validation HTTP middleware for any kind-schematized object type.
type Base struct {
	all  []kindsys.Core
	test *test.Kind
}

// type guards
var (
	_ kindsys.Core = &test.Kind{}
)

// Test returns the [kindsys.Interface] implementation for the test kind.
func (b *Base) Test() *test.Kind {
	return b.test
}

func doNewBase(rt *thema.Runtime) *Base {
	var err error
	reg := &Base{}

	reg.test, err = test.NewKind(rt)
	if err != nil {
		panic(fmt.Sprintf("error while initializing the test Kind: %s", err))
	}
	reg.all = append(reg.all, reg.test)

	return reg
}
