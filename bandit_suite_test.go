package bandit_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func XTestBandit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bandit Suite")
}
