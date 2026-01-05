package cache_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/cache"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"
)

var (
	mockCtrl  *gomock.Controller
	mockStore *MockStore
)

func TestUnitCache(t *testing.T) {
	spec.Run(t, "Testing the history store", testCache, spec.Report(report.Terminal{}))
}

func testCache(t *testing.T, when spec.G, it spec.S) {
	var subject *cache.Cache

	const endpoint = "https://www.endpoint.com/mcp"

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockStore = NewMockStore(mockCtrl)
		subject = cache.New(mockStore)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("GetSessionID()", func() {
		it("should throw an error when the store returns an error", func() {
			const msg = "error-message"
			mockStore.EXPECT().Get(gomock.Any()).Return(nil, errors.New(msg))

			_, err := subject.GetSessionID(endpoint)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})
		it("should throw an error when the store returns an invalid json", func() {
			const invalid = `{"no-closing":"bracket"`
			mockStore.EXPECT().Get(gomock.Any()).Return([]byte(invalid), nil)

			_, err := subject.GetSessionID(endpoint)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("unexpected end of JSON input"))
		})
		it("returns the expected result when there are no errors", func() {
			const sessionId = "session_id"
			json := fmt.Sprintf(`{"endpoint": "%s", "session_id": "%s"}`, endpoint, sessionId)

			mockStore.EXPECT().
				Get(gomock.Any()).
				DoAndReturn(func(key string) ([]byte, error) {
					// verify we hash the endpoint
					Expect(key).ToNot(Equal(endpoint))
					Expect(len(key)).To(Equal(64)) // sha256 hex length
					return []byte(json), nil
				})

			result, err := subject.GetSessionID(endpoint)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(sessionId))
		})
	})
	when("SetSessionID()", func() {
		it("writes a JSON entry to the store under the hashed endpoint key", func() {
			const sessionId = "session_id"

			mockStore.EXPECT().
				Set(gomock.Any(), gomock.Any()).
				DoAndReturn(func(key string, raw []byte) error {
					// verify key looks like sha256(endpoint)
					Expect(key).ToNot(Equal(endpoint))
					Expect(len(key)).To(Equal(64))

					// verify payload
					var e cache.Entry
					Expect(json.Unmarshal(raw, &e)).To(Succeed())
					Expect(e.Endpoint).To(Equal(endpoint))
					Expect(e.SessionID).To(Equal(sessionId))
					Expect(e.UpdatedAt.IsZero()).To(BeFalse())

					return nil
				})

			err := subject.SetSessionID(endpoint, sessionId)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns an error when the store Set fails", func() {
			const sessionId = "session_id"
			const msg = "set failed"

			mockStore.EXPECT().
				Set(gomock.Any(), gomock.Any()).
				Return(errors.New(msg))

			err := subject.SetSessionID(endpoint, sessionId)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})
	})
	when("DeleteSessionID()", func() {
		it("deletes the hashed endpoint key from the store", func() {
			mockStore.EXPECT().
				Delete(gomock.Any()).
				DoAndReturn(func(key string) error {
					Expect(key).ToNot(Equal(endpoint))
					Expect(len(key)).To(Equal(64))
					return nil
				})

			err := subject.DeleteSessionID(endpoint)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns an error when the store Delete fails", func() {
			const msg = "delete failed"
			mockStore.EXPECT().Delete(gomock.Any()).Return(errors.New(msg))

			err := subject.DeleteSessionID(endpoint)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(msg))
		})
	})
}
