package main

import (
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
	. "github.com/onsi/gomega"
)

func TestFilterSliceElemsWithTest(t *testing.T) {
	g := NewGomegaWithT(t)

	crdNameContainsSmiTestPredicate := func(s string) bool { return strings.Contains(s, "smi-spec.io") }
	testSlice := []string{"abc", "httproutegroups.specs.smi-spec.io", "def", "tcproutes.specs.smi-spec.io", "ghi"}
	expectedSlice := []string{"httproutegroups.specs.smi-spec.io", "tcproutes.specs.smi-spec.io"}
	actualSlice := utils.FilterSliceElemsWithTestPredicate(testSlice, crdNameContainsSmiTestPredicate)

	g.Expect(actualSlice).To(Equal(expectedSlice))
}
