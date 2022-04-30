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
	code := m.Run()
	if code != 0 {
		fixture.PrintDiagnostics()
	}

	// Check our tests haven't resulted in any unexpected Docker image usage
	err := test.CheckDockerImages(fixture.Clientset)
	if err != nil {
		log.Printf("Failing due to unexpected Docker image usage (see test.dockerImageManager): %v", err)
		code = 1
	}

	return code
}
