package reexecutable_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReexecutable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reexecutable Suite")
}
