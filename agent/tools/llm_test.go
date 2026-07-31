package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kardolus/chatgpt-cli/agent/tools"
	"github.com/kardolus/chatgpt-cli/api"
	apiclient "github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

// fakeCaller implements http.Caller; only Post is exercised by Query.
type fakeCaller struct {
	postResp []byte
	postErr  error
}

func (f *fakeCaller) Post(_ context.Context, _ string, _ []byte, _ bool) ([]byte, error) {
	return f.postResp, f.postErr
}
func (f *fakeCaller) PostWithHeaders(_ context.Context, _ string, _ []byte, _ map[string]string) ([]byte, error) {
	return f.postResp, f.postErr
}
func (f *fakeCaller) Get(_ context.Context, _ string) ([]byte, error) { return nil, nil }
func (f *fakeCaller) PostWithHeadersResponse(_ context.Context, _ string, _ []byte, _ map[string]string) (api.HTTPResponse, error) {
	return api.HTTPResponse{}, nil
}

// recordingStore implements history.Store and records whether it was touched.
// Complete() must set OmitHistory=true, so a correctly-behaving wrapper never
// reads or writes history during the call.
type recordingStore struct {
	readCalled  bool
	writeCalled bool
}

func (s *recordingStore) Read() ([]history.History, error)             { s.readCalled = true; return nil, nil }
func (s *recordingStore) ReadThread(string) ([]history.History, error) { return nil, nil }
func (s *recordingStore) Write([]history.History) error                { s.writeCalled = true; return nil }
func (s *recordingStore) SetThread(string)                             {}
func (s *recordingStore) GetThread() string                            { return "" }

type fixedTimer struct{}

func (fixedTimer) Now() time.Time { return time.Unix(0, 0) }

func newLLMTestClient(caller http.Caller, store history.Store) *apiclient.Client {
	cfg := config.Config{Model: "gpt-4o", OmitHistory: false, Temperature: 0.7}
	return apiclient.New(
		func(config.Config) http.Caller { return caller },
		store,
		fixedTimer{},
		fsio.NewRealReader(fsio.DefaultBufferSize),
		&fsio.RealWriter{},
		cfg,
	)
}

func TestUnitClientLLM(t *testing.T) {
	spec.Run(t, "Testing the ClientLLM wrapper", testClientLLM, spec.Report(report.Terminal{}))
}

func testClientLLM(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("Complete()", func() {
		it("restores config and leaves history untouched on success", func() {
			resp := []byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}],"usage":{"total_tokens":5}}`)
			store := &recordingStore{}
			c := newLLMTestClient(&fakeCaller{postResp: resp}, store)

			out, _, err := tools.NewClientLLM(c).Complete(context.Background(), "hi")

			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("hello"))
			// OmitHistory was set true during the call -> store never touched.
			Expect(store.readCalled).To(BeFalse())
			Expect(store.writeCalled).To(BeFalse())
			// Config restored to its pre-call values.
			Expect(c.Config.OmitHistory).To(BeFalse())
			Expect(c.Config.Temperature).To(Equal(0.7))
		})

		it("restores config even when the query fails", func() {
			store := &recordingStore{}
			c := newLLMTestClient(&fakeCaller{postErr: errors.New("boom")}, store)

			_, _, err := tools.NewClientLLM(c).Complete(context.Background(), "hi")

			Expect(err).To(HaveOccurred())
			Expect(c.Config.OmitHistory).To(BeFalse())
			Expect(c.Config.Temperature).To(Equal(0.7))
		})
	})
}
