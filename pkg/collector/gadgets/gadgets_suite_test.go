package gadgets_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGadgets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gadgets Suite")
}
