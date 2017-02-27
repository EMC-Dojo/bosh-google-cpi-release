package accelerator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAcceleratorService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AcceleratorService Suite")
}
