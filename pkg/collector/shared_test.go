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
		log.Fatalf("Error initializing tests: %v", err)
	}
	code := m.Run()
	fixture.Cleanup()
	os.Exit(code)
}
