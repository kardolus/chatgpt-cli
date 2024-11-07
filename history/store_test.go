package history_test

import (
	"github.com/kardolus/chatgpt-cli/history"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"
)

func TestUnitStore(t *testing.T) {
	spec.Run(t, "Testing the history store", testStore, spec.Report(report.Terminal{}))
}

func testStore(t *testing.T, when spec.G, it spec.S) {
	var subject history.Store

	it.Before(func() {
		RegisterTestingT(t)
		subject = &history.FileIO{}
	})

	when("GetThread()", func() {
		it("should return the thread", func() {
			thread := "threadName"
			subject.SetThread(thread)
			Expect(subject.GetThread()).To(Equal(thread))
		})
	})
}
