package gadgets_test

import (
	"github.com/Azure/aks-periscope/pkg/collector/gadgets"
	"github.com/Azure/aks-periscope/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"log"
	"os/exec"
	"time"
)

var _ = Describe("DnsTracerCollector", func() {
	It("should have collector name set to dns-tracer", func() {
		const expectedName = "dns-tracer"

		c := gadgets.NewDNSTracerCollector(utils.Linux)
		actualName := c.GetName()
		Expect(actualName).To(Equal(expectedName))
	})

	DescribeTable("tracer works with linux only", func(osIdentifier utils.OSIdentifier, supported bool) {
		err := gadgets.NewDNSTracerCollector(osIdentifier).CheckSupported()
		if supported {
			Expect(err).To(BeNil())
		} else {
			Expect(err).ToNot(BeNil())
		}
	},
		Entry("not supported on windows nodes", utils.Windows, false),
		Entry("supported on linux nodes", utils.Linux, true),
	)

	Context("collect dns trace on linux node", func() {
		dnsTracerCollector := gadgets.NewDNSTracerCollector(utils.Linux)
		if err := dnsTracerCollector.CheckSupported(); err != nil {
			Fail("DNS tracer is not supported")
		}

		done := make(chan struct{})

		By("Starting tracer ")

		go func() {
			err := dnsTracerCollector.Collect()
			if err != nil {
				Fail("could not collect dns trace information")
			}
			close(done)
		}()

		By("Starting DNS activities")
		urls := []string{"microsoft.com.", "google.com.", "shouldnotexist.com."}
		i := 0
	loop:
		for {
			select {
			case <-done:
				log.Printf("Stopping")
				break loop

			default:
				log.Printf("running nslookup %v\n", urls[i])
				nslookup := exec.Command("nslookup", "-querytype=a", urls[i])
				_, err := nslookup.Output()
				if err != nil {
					Fail("Could not exec nslookup " + urls[i])
				}
				i = (i + 1) % len(urls)
				time.Sleep(2 * time.Second)
			}
		}

		By("Collecting result")
		data := dnsTracerCollector.GetData()
		log.Printf("==========================\nCollected dns trace data %v \n ==========================\n", data)
		Expect(data).ToNot(BeNil())
	})
})
