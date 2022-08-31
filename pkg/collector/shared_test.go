package collector

import (
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
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
	err := fixture.CheckDockerImages()
	if err != nil {
		// Fail the test run (even if the actual tests passed) to avoid merging code that
		// pulls images during tests.
		log.Printf("Failing due to unexpected Docker image usage (see test.dockerImageManager): %v", err)
		code = 1
	}

	return code
}

func testDataValue(t *testing.T, dataValue interfaces.DataValue, test func(string)) {
	value, err := utils.GetContent(func() (io.ReadCloser, error) { return dataValue.GetReader() })
	if err != nil {
		t.Errorf("error reading value: %v", err)
	}
	test(value)
}

func compareCollectorData(t *testing.T, expectedData map[string]*regexp.Regexp, actualData map[string]interfaces.DataValue) {
	missingDataKeys := []string{}
	for key, regexp := range expectedData {
		dataValue, ok := actualData[key]
		if ok {
			testDataValue(t, dataValue, func(value string) {
				if !regexp.MatchString(value) {
					t.Errorf("unexpected value for %s\n\texpected: %s\n\tfound: %s", key, regexp.String(), value)
				}
			})
		} else {
			missingDataKeys = append(missingDataKeys, key)
		}
	}
	if len(missingDataKeys) > 0 {
		t.Errorf("missing keys in actual data:\n%s", strings.Join(missingDataKeys, "\n"))
	}

	unexpectedDataKeys := []string{}
	for key := range actualData {
		if _, ok := expectedData[key]; !ok {
			unexpectedDataKeys = append(unexpectedDataKeys, key)
		}
	}
	if len(unexpectedDataKeys) > 0 {
		t.Errorf("unexpected keys in actual data:\n%s", strings.Join(unexpectedDataKeys, "\n"))
	}
}
