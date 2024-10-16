package history_test

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/types"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"
)

//go:generate mockgen -destination=historymocks_test.go -package=history_test github.com/kardolus/chatgpt-cli/history HistoryStore

var (
	mockCtrl         *gomock.Controller
	mockHistoryStore *MockHistoryStore
	subject          *history.History
)

func TestUnitHistory(t *testing.T) {
	spec.Run(t, "Testing the History", testHistory, spec.Report(report.Terminal{}))
}

func testHistory(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockHistoryStore = NewMockHistoryStore(mockCtrl)
		subject = history.NewHistory(mockHistoryStore)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("Print()", func() {
		const threadName = "threadName"

		it("throws an error when there is a problem talking to the store", func() {
			mockHistoryStore.EXPECT().ReadThread(threadName).Return(nil, errors.New("nope")).Times(1)

			_, err := subject.Print(threadName)
			Expect(err).To(HaveOccurred())
		})

		it("concatenates multiple user messages", func() {
			messages := []types.Message{
				{Role: "user", Content: "first message"},
				{Role: "user", Content: " second message"},
				{Role: "assistant", Content: "response"},
			}

			mockHistoryStore.EXPECT().ReadThread(threadName).Return(messages, nil).Times(1)

			result, err := subject.Print(threadName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("**USER** 👤:\nfirst message second message\n"))
			Expect(result).To(ContainSubstring("**ASSISTANT** 🤖:\nresponse\n"))
		})

		it("prints all roles correctly", func() {
			messages := []types.Message{
				{Role: "system", Content: "system message"},
				{Role: "user", Content: "user message"},
				{Role: "assistant", Content: "assistant message"},
			}

			mockHistoryStore.EXPECT().ReadThread(threadName).Return(messages, nil).Times(1)

			result, err := subject.Print(threadName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("**SYSTEM** 💻:\nsystem message\n"))
			Expect(result).To(ContainSubstring("\n---\n**USER** 👤:\nuser message\n"))
			Expect(result).To(ContainSubstring("**ASSISTANT** 🤖:\nassistant message\n"))
		})

		it("handles the final user message concatenation", func() {
			messages := []types.Message{
				{Role: "user", Content: "first message"},
				{Role: "user", Content: " second message"},
			}

			mockHistoryStore.EXPECT().ReadThread(threadName).Return(messages, nil).Times(1)

			result, err := subject.Print(threadName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("**USER** 👤:\nfirst message second message\n"))
		})
	})
}
