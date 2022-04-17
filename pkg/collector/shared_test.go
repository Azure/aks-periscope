package collector

import (
	"log"
	"os"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
)

func TestMain(m *testing.M) {
	fixture, err := test.GetClusterFixture()
	if err != nil {
		fixture.Cleanup()
		log.Fatalf("Error initializing tests: %v", err)
	}
	code := runTests(m, fixture)
	os.Exit(code)
}

func runTests(m *testing.M, fixture *test.ClusterFixture) int {
	defer fixture.Cleanup()
	return m.Run()
}
