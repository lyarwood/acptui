package ambient_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAmbient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ambient Suite")
}
