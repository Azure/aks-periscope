package collector

import (
	"log"
	"os"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
)

// TestMain coordinates the execution of all tests in the package. This is required because they all share
// common initialization and cleanup code.
func TestMain(m *testing.M) {
	fixture, err := test.GetClusterFixture()
	if err != nil {
		// Initialization failed, so clean up and exit before even running tests.
		fixture.Cleanup()
		log.Fatalf("Error initializing tests: %v", err)
	}
	code := runTests(m, fixture)
	os.Exit(code)
}

func runTests(m *testing.M, fixture *test.ClusterFixture) int {
	// Always clean up after running all the tests. This is not strictly necessary,
	// but helps ensure a clean test cluster for subsequent local test runs.
	defer fixture.Cleanup()

	// Run all the tests in the package.
	code := m.Run()
	if code != 0 {
		// Output some informtation that may help diagnose test failures.
		fixture.PrintDiagnostics()
	}

	// Check our tests haven't resulted in any unexpected Docker image usage
	err := test.CheckDockerImages(fixture.Clientset)
	if err != nil {
		// Fail the test run (even if the actual tests passed) to avoid merging code that
		// pulls images during tests.
		log.Printf("Failing due to unexpected Docker image usage (see test.dockerImageManager): %v", err)
		code = 1
	}

	return code
}
