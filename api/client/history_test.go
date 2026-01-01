// history_test.go
package client_test

import (
	"testing"
	"time"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/history"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testHistory(t *testing.T, when spec.G, it spec.S) {
	when("History()", func() {
		when("ProvideContext()", func() {
			it("updates the history with the provided context", func() {
				subject := factory.buildClientWithoutConfig()

				chatContext := "This is a story about a dog named Kya. Kya loves to play fetch and swim in the lake."
				mockHistoryStore.EXPECT().Read().Return(nil, nil).Times(1)

				mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

				subject.ProvideContext(chatContext)

				Expect(len(subject.History)).To(Equal(2)) // system message + provided context

				systemMessage := subject.History[0]
				Expect(systemMessage.Role).To(Equal(client.SystemRole))
				Expect(systemMessage.Content).To(Equal(config.Role))

				contextMessage := subject.History[1]
				Expect(contextMessage.Role).To(Equal(client.UserRole))
				Expect(contextMessage.Content).To(Equal(chatContext))
			})

			it("behaves as expected with a non empty initial history", func() {
				subject := factory.buildClientWithoutConfig()

				subject.History = []history.History{
					{
						Message: api.Message{
							Role:    client.SystemRole,
							Content: "system message",
						},
					},
					{
						Message: api.Message{
							Role: client.UserRole,
						},
					},
				}

				mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

				chatContext := "test context"
				subject.ProvideContext(chatContext)

				Expect(len(subject.History)).To(Equal(3))

				contextMessage := subject.History[2]
				Expect(contextMessage.Role).To(Equal(client.UserRole))
				Expect(contextMessage.Content).To(Equal(chatContext))
			})
		})
	})
}
