package load_test

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/api7/gopkg/pkg/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var closer io.Closer

func init() {
	// save log locally
	file, err := os.Create(time.Now().Format("load_test_200601021504.log"))
	if err != nil {
		log.Fatalf("failed to create log file, err: %v", err)
	}
	closer = file
	GinkgoWriter.TeeTo(file)
}

// Run long-term-stability tests using Ginkgo runner.
func TestLongTermStability(t *testing.T) {
	defer func() { _ = closer.Close() }()

	RegisterFailHandler(Fail)
	var f = framework.NewFramework()
	_ = f

	scaffold.NewDeployer = func(s *scaffold.Scaffold) scaffold.Deployer {
		return scaffold.NewAPISIXDeployer(s)
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting load-test suite\n")
	RunSpecs(t, "long-term-stability suite")
}
