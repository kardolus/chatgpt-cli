// media_test.go
package client_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/history"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testMedia(t *testing.T, when spec.G, it spec.S) {
	when("Media()", func() {
		when("SynthesizeSpeech()", func() {
			const (
				inputText      = "mock-input"
				outputFile     = "mock-output"
				outputFileType = "mp3"
				errorText      = "mock error occurred"
			)

			var (
				subject  *client.Client
				fileName = outputFile + "." + outputFileType
				body     []byte
				response []byte
			)

			it.Before(func() {
				subject = factory.buildClientWithoutConfig()
				request := api.Speech{
					Model:          subject.Config.Model,
					Voice:          subject.Config.Voice,
					Input:          inputText,
					ResponseFormat: outputFileType,
				}
				var err error
				body, err = json.Marshal(request)
				Expect(err).NotTo(HaveOccurred())

				response = []byte("mock response")
			})

			it("throws an error when the http call fails", func() {
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.SpeechPath, body, false).
					Return(nil, errors.New(errorText))

				err := subject.SynthesizeSpeech(inputText, fileName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorText))
			})

			it("throws an error when a file cannot be created", func() {
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.SpeechPath, body, false).
					Return(response, nil)
				mockWriter.EXPECT().Create(fileName).Return(nil, errors.New(errorText))

				err := subject.SynthesizeSpeech(inputText, fileName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorText))
			})

			it("throws an error when bytes cannot be written to the output file", func() {
				file, err := os.Open(os.DevNull)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.SpeechPath, body, false).
					Return(response, nil)
				mockWriter.EXPECT().Create(fileName).Return(file, nil)
				mockWriter.EXPECT().Write(file, response).Return(errors.New(errorText))

				err = subject.SynthesizeSpeech(inputText, fileName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorText))
			})

			it("succeeds when no errors occurred", func() {
				file, err := os.Open(os.DevNull)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.SpeechPath, body, false).
					Return(response, nil)
				mockWriter.EXPECT().Create(fileName).Return(file, nil)
				mockWriter.EXPECT().Write(file, response).Return(nil)

				err = subject.SynthesizeSpeech(inputText, fileName)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		when("GenerateImage()", func() {
			const (
				inputText  = "draw a happy dog"
				outputFile = "dog.png"
				errorText  = "mock error occurred"
			)

			var (
				subject *client.Client
				body    []byte
			)

			it.Before(func() {
				subject = factory.buildClientWithoutConfig()
				request := api.Draw{
					Model:  subject.Config.Model,
					Prompt: inputText,
				}
				var err error
				body, err = json.Marshal(request)
				Expect(err).NotTo(HaveOccurred())
			})

			it("throws an error when the http call fails", func() {
				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.ImageGenerationsPath, body, false).
					Return(nil, errors.New(errorText))

				err := subject.GenerateImage(inputText, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorText))
			})

			it("throws an error when no image data is returned", func() {
				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.ImageGenerationsPath, body, false).
					Return([]byte(`{"data":[]}`), nil)

				err := subject.GenerateImage(inputText, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no image data returned"))
			})

			it("throws an error when base64 is invalid", func() {
				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.ImageGenerationsPath, body, false).
					Return([]byte(`{"data":[{"b64_json":"!!notbase64!!"}]}`), nil)

				err := subject.GenerateImage(inputText, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to decode base64 image"))
			})

			it("throws an error when a file cannot be created", func() {
				valid := base64.StdEncoding.EncodeToString([]byte("image-bytes"))

				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.ImageGenerationsPath, body, false).
					Return([]byte(fmt.Sprintf(`{"data":[{"b64_json":"%s"}]}`, valid)), nil)

				mockWriter.EXPECT().Create(outputFile).Return(nil, errors.New(errorText))

				err := subject.GenerateImage(inputText, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorText))
			})

			it("throws an error when bytes cannot be written to the file", func() {
				valid := base64.StdEncoding.EncodeToString([]byte("image-bytes"))
				file, err := os.Open(os.DevNull)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.ImageGenerationsPath, body, false).
					Return([]byte(fmt.Sprintf(`{"data":[{"b64_json":"%s"}]}`, valid)), nil)

				mockWriter.EXPECT().Create(outputFile).Return(file, nil)
				mockWriter.EXPECT().Write(file, []byte("image-bytes")).Return(errors.New(errorText))

				err = subject.GenerateImage(inputText, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorText))
			})

			it("succeeds when all steps complete", func() {
				valid := base64.StdEncoding.EncodeToString([]byte("image-bytes"))
				file, err := os.Open(os.DevNull)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.ImageGenerationsPath, body, false).
					Return([]byte(fmt.Sprintf(`{"data":[{"b64_json":"%s"}]}`, valid)), nil)

				mockWriter.EXPECT().Create(outputFile).Return(file, nil)
				mockWriter.EXPECT().Write(file, []byte("image-bytes")).Return(nil)

				err = subject.GenerateImage(inputText, outputFile)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		when("EditImage()", func() {
			const (
				inputText  = "give the dog sunglasses"
				inputFile  = "dog.png"
				outputFile = "dog_cool.png"
				errorText  = "mock error occurred"
			)

			var (
				subject    *client.Client
				validB64   string
				imageBytes = []byte("image-bytes")
				respBytes  []byte
			)

			it.Before(func() {
				subject = factory.buildClientWithoutConfig()
				validB64 = base64.StdEncoding.EncodeToString(imageBytes)
				respBytes = []byte(fmt.Sprintf(`{"data":[{"b64_json":"%s"}]}`, validB64))
			})

			it("returns error when input file can't be opened", func() {
				mockReader.EXPECT().Open(inputFile).Return(nil, errors.New(errorText))

				err := subject.EditImage(inputText, inputFile, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to open input image"))
			})

			it("returns error on invalid mime type", func() {
				file := openDummy()
				mockReader.EXPECT().Open(inputFile).Return(file, nil).Times(2)
				mockReader.EXPECT().ReadBufferFromFile(file).Return([]byte("not an image"), nil)

				err := subject.EditImage(inputText, inputFile, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported MIME type"))
			})

			it("returns error when HTTP call fails", func() {
				mockReader.EXPECT().Open(inputFile).DoAndReturn(func(string) (*os.File, error) {
					return openDummy(), nil
				}).Times(2)

				mockReader.EXPECT().
					ReadBufferFromFile(gomock.AssignableToTypeOf(&os.File{})).
					Return([]byte("\x89PNG\r\n\x1a\n"), nil)

				mockCaller.EXPECT().
					PostWithHeaders(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New(errorText))

				err := subject.EditImage(inputText, inputFile, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to edit image"))
			})

			it("returns error when base64 is invalid", func() {
				invalidResp := []byte(`{"data":[{"b64_json":"!notbase64"}]}`)

				mockReader.EXPECT().Open(inputFile).DoAndReturn(func(string) (*os.File, error) {
					return openDummy(), nil
				}).Times(2)

				mockReader.EXPECT().
					ReadBufferFromFile(gomock.AssignableToTypeOf(&os.File{})).
					Return([]byte("\x89PNG\r\n\x1a\n"), nil)

				mockCaller.EXPECT().
					PostWithHeaders(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(invalidResp, nil)

				err := subject.EditImage(inputText, inputFile, outputFile)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to decode base64 image"))
			})

			it("writes image when all steps succeed", func() {
				file := openDummy()
				mockReader.EXPECT().Open(inputFile).DoAndReturn(func(string) (*os.File, error) {
					return openDummy(), nil
				}).Times(2)

				mockReader.EXPECT().
					ReadBufferFromFile(gomock.AssignableToTypeOf(&os.File{})).
					Return([]byte("\x89PNG\r\n\x1a\n"), nil)

				mockCaller.EXPECT().
					PostWithHeaders(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(respBytes, nil)

				mockWriter.EXPECT().Create(outputFile).Return(file, nil)
				mockWriter.EXPECT().Write(file, imageBytes).Return(nil)

				err := subject.EditImage(inputText, inputFile, outputFile)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		when("Transcribe()", func() {
			const audioPath = "path/to/audio.wav"
			const transcribedText = "Hello, this is a test."

			it("returns an error if the audio file cannot be opened", func() {
				subject := factory.buildClientWithoutConfig()

				mockHistoryStore.EXPECT().Read().Return(nil, nil)
				mockTimer.EXPECT().Now().Times(1)

				mockReader.EXPECT().Open(audioPath).Return(nil, errors.New("cannot open"))

				_, err := subject.Transcribe(audioPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot open"))
			})

			it("returns an error if copying audio content fails", func() {
				subject := factory.buildClientWithoutConfig()

				mockHistoryStore.EXPECT().Read().Return(nil, nil)
				mockTimer.EXPECT().Now().Times(1)

				reader, writer, err := os.Pipe()
				Expect(err).NotTo(HaveOccurred())
				_ = writer.Close() // force EOF/copy failure behavior

				mockReader.EXPECT().Open(audioPath).Return(reader, nil)

				mockCaller.EXPECT().
					PostWithHeaders(subject.Config.URL+subject.Config.TranscriptionsPath, gomock.Any(), gomock.Any())

				_, err = subject.Transcribe(audioPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed"))
			})

			it("returns an error if the API call fails", func() {
				subject := factory.buildClientWithoutConfig()

				mockHistoryStore.EXPECT().Read().Return(nil, nil)
				mockTimer.EXPECT().Now().Times(1)

				file, err := os.Open(os.DevNull)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				mockReader.EXPECT().Open(audioPath).Return(file, nil)

				mockCaller.EXPECT().
					PostWithHeaders(subject.Config.URL+subject.Config.TranscriptionsPath, gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error"))

				_, err = subject.Transcribe(audioPath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("network error"))
			})

			it("returns the transcribed text when successful", func() {
				subject := factory.buildClientWithoutConfig()

				mockHistoryStore.EXPECT().Read().Return(nil, nil)

				now := time.Now()
				mockTimer.EXPECT().Now().Return(now).Times(3)

				file, err := os.Open(os.DevNull)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				mockReader.EXPECT().Open(audioPath).Return(file, nil)

				resp := []byte(`{"text": "Hello, this is a test."}`)
				mockCaller.EXPECT().
					PostWithHeaders(subject.Config.URL+subject.Config.TranscriptionsPath, gomock.Any(), gomock.Any()).
					Return(resp, nil)

				expectedHistory := []history.History{
					{
						Message: api.Message{
							Role:    client.SystemRole,
							Content: subject.Config.Role,
						},
						Timestamp: now,
					},
					{
						Message: api.Message{
							Role:    client.UserRole,
							Content: "[transcribe] audio.wav",
						},
						Timestamp: now,
					},
					{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: transcribedText,
						},
						Timestamp: now,
					},
				}

				mockHistoryStore.EXPECT().Write(expectedHistory)

				text, err := subject.Transcribe(audioPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(text).To(Equal(transcribedText))
			})
		})
	})
}

func openDummy() *os.File {
	// Use os.Pipe to get an *os.File without needing a real disk file.
	r, w, _ := os.Pipe()
	go func() {
		_, _ = io.Copy(w, bytes.NewBuffer([]byte("\x89PNG\r\n\x1a\n")))
		_ = w.Close()
	}()
	return r
}
