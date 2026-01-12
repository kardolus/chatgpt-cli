package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal"
	"io"
	"mime/multipart"
	stdhttp "net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	audioType    = "input_audio"
	imageContent = "data:%s;base64,%s"
	imageURLType = "image_url"
	httpScheme   = "http"
	httpsScheme  = "https"
)

// EditImage edits an input image using a text prompt and writes the modified image to the specified output path.
//
// This method sends a multipart/form-data POST request to the image editing endpoint
// (typically OpenAI's /v1/images/edits). The request includes:
//   - The image file to edit.
//   - A text prompt describing how the image should be modified.
//   - The model ID (e.g., gpt-image-1).
//
// The response is expected to contain a base64-encoded image, which is decoded and written to the outputPath.
//
// Parameters:
//   - inputText: A text prompt describing the desired modifications to the image.
//   - inputPath: The file path to the source image (must be a supported format: PNG, JPEG, or WebP).
//   - outputPath: The file path where the edited image will be saved.
//
// Returns:
//   - An error if any step of the process fails: reading the file, building the request, sending it,
//     decoding the response, or writing the output image.
//
// Example:
//
//	err := client.EditImage("Add a rainbow in the sky", "input.png", "output.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) EditImage(inputText, inputPath, outputPath string) error {
	endpoint := c.getEndpoint(c.Config.ImageEditsPath)

	file, err := c.reader.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input image: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	mimeType, err := c.getMimeTypeFromFileContent(inputPath)
	if err != nil {
		return fmt.Errorf("failed to detect MIME type: %w", err)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return fmt.Errorf("unsupported MIME type: %s", mimeType)
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, filepath.Base(inputPath)))
	header.Set("Content-Type", mimeType)

	part, err := writer.CreatePart(header)
	if err != nil {
		return fmt.Errorf("failed to create image part: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy image data: %w", err)
	}

	if err := writer.WriteField("prompt", inputText); err != nil {
		return fmt.Errorf("failed to add prompt: %w", err)
	}
	if err := writer.WriteField("model", c.Config.Model); err != nil {
		return fmt.Errorf("failed to add model: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	c.printRequestDebugInfo(endpoint, buf.Bytes(), map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})

	respBytes, err := c.Caller.PostWithHeaders(endpoint, buf.Bytes(), map[string]string{
		c.Config.AuthHeader:           fmt.Sprintf("%s %s", c.Config.AuthTokenPrefix, c.Config.APIKey),
		internal.HeaderContentTypeKey: writer.FormDataContentType(),
	})
	if err != nil {
		return fmt.Errorf("failed to edit image: %w", err)
	}

	// Parse the JSON and extract b64_json
	var response struct {
		Data []struct {
			B64 string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	if len(response.Data) == 0 {
		return fmt.Errorf("no image data returned")
	}

	imgBytes, err := base64.StdEncoding.DecodeString(response.Data[0].B64)
	if err != nil {
		return fmt.Errorf("failed to decode base64 image: %w", err)
	}

	outFile, err := c.writer.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := c.writer.Write(outFile, imgBytes); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	c.printResponseDebugInfo([]byte(fmt.Sprintf("[image] %d bytes written to %s", len(imgBytes), outputPath)))
	return nil
}

// GenerateImage sends a prompt to the configured image generation model (e.g., gpt-image-1)
// and writes the resulting image to the specified output path.
//
// The method performs the following steps:
//  1. Sends a POST request to the image generation endpoint with the provided prompt.
//  2. Parses the response and extracts the base64-encoded image data.
//  3. Decodes the image bytes and writes them to the given outputPath.
//  4. Logs the number of bytes written using debug output.
//
// Parameters:
//   - inputText: The prompt describing the image to be generated.
//   - outputPath: The file path where the generated image (e.g., .png) will be saved.
//
// Returns:
//   - An error if any part of the request, decoding, or file writing fails.
func (c *Client) GenerateImage(inputText, outputPath string) error {
	req := api.Draw{
		Model:  c.Config.Model,
		Prompt: inputText,
	}

	return c.postAndWriteBinaryOutput(
		c.getEndpoint(c.Config.ImageGenerationsPath),
		req,
		outputPath,
		"image",
		func(respBytes []byte) ([]byte, error) {
			var response struct {
				Data []struct {
					B64 string `json:"b64_json"`
				} `json:"data"`
			}
			if err := json.Unmarshal(respBytes, &response); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			if len(response.Data) == 0 {
				return nil, fmt.Errorf("no image data returned")
			}
			decoded, err := base64.StdEncoding.DecodeString(response.Data[0].B64)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 image: %w", err)
			}
			return decoded, nil
		},
	)
}

// SynthesizeSpeech converts the given input text into speech using the configured TTS model,
// and writes the resulting audio to the specified output file.
//
// The audio format is inferred from the output file's extension (e.g., "mp3", "wav") and sent
// as the "response_format" in the request to the OpenAI speech synthesis endpoint.
//
// Parameters:
//   - inputText: The text to synthesize into speech.
//   - outputPath: The path to the output audio file. The file extension determines the response format.
//
// Returns an error if the request fails, the response cannot be written, or the file cannot be created.
func (c *Client) SynthesizeSpeech(inputText, outputPath string) error {
	req := api.Speech{
		Model:          c.Config.Model,
		Voice:          c.Config.Voice,
		Input:          inputText,
		ResponseFormat: getExtension(outputPath),
	}
	return c.postAndWriteBinaryOutput(c.getEndpoint(c.Config.SpeechPath), req, outputPath, "binary", nil)
}

// Transcribe uploads an audio file to the OpenAI transcription endpoint and returns the transcribed text.
//
// It reads the audio file from the provided `audioPath`, creates a multipart/form-data request with the model name
// and the audio file, and sends it to the endpoint defined by the `TranscriptionsPath` in the client config.
// The method expects a JSON response containing a "text" field with the transcription result.
//
// Parameters:
//   - audioPath: The local file path to the audio file to be transcribed.
//
// Returns:
//   - string: The transcribed text from the audio file.
//   - error: An error if the file can't be read, the request fails, or the response is invalid.
//
// This method supports formats like mp3, mp4, mpeg, mpga, m4a, wav, and webm, depending on API compatibility.
func (c *Client) Transcribe(audioPath string) (string, error) {
	c.initHistory()

	file, err := c.reader.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("model", c.Config.Model)

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	endpoint := c.getEndpoint(c.Config.TranscriptionsPath)
	headers := map[string]string{
		internal.HeaderContentTypeKey: writer.FormDataContentType(),
		c.Config.AuthHeader:           fmt.Sprintf("%s %s", c.Config.AuthTokenPrefix, c.Config.APIKey),
	}

	c.printRequestDebugInfo(endpoint, buf.Bytes(), headers)

	raw, err := c.Caller.PostWithHeaders(endpoint, buf.Bytes(), headers)
	if err != nil {
		return "", err
	}

	c.printResponseDebugInfo(raw)

	var res struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("failed to parse transcription: %w", err)
	}

	c.History = append(c.History, history.History{
		Message: api.Message{
			Role:    UserRole,
			Content: fmt.Sprintf("[transcribe] %s", filepath.Base(audioPath)),
		},
		Timestamp: c.timer.Now(),
	})

	c.History = append(c.History, history.History{
		Message: api.Message{
			Role:    AssistantRole,
			Content: res.Text,
		},
		Timestamp: c.timer.Now(),
	})

	c.truncateHistory()

	if !c.Config.OmitHistory {
		_ = c.historyStore.Write(c.History)
	}

	return res.Text, nil
}

func (c *Client) appendMediaMessages(ctx context.Context, messages []api.Message) ([]api.Message, error) {
	if data, ok := ctx.Value(internal.BinaryDataKey).([]byte); ok {
		content, err := c.createImageContentFromBinary(data)
		if err != nil {
			return nil, err
		}
		messages = append(messages, api.Message{
			Role:    UserRole,
			Content: []api.ImageContent{content},
		})
	} else if path, ok := ctx.Value(internal.ImagePathKey).(string); ok {
		content, err := c.createImageContentFromURLOrFile(path)
		if err != nil {
			return nil, err
		}
		messages = append(messages, api.Message{
			Role:    UserRole,
			Content: []api.ImageContent{content},
		})
	} else if path, ok := ctx.Value(internal.AudioPathKey).(string); ok {
		content, err := c.createAudioContentFromFile(path)
		if err != nil {
			return nil, err
		}
		messages = append(messages, api.Message{
			Role:    UserRole,
			Content: []api.AudioContent{content},
		})
	}
	return messages, nil
}

func (c *Client) base64Encode(path string) (string, error) {
	imageData, err := c.reader.ReadFile(path)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(imageData), nil
}

func (c *Client) createAudioContentFromFile(audio string) (api.AudioContent, error) {

	format, err := c.detectAudioFormat(audio)
	if err != nil {
		return api.AudioContent{}, err
	}

	encodedAudio, err := c.base64Encode(audio)
	if err != nil {
		return api.AudioContent{}, err
	}

	return api.AudioContent{
		Type: audioType,
		InputAudio: api.InputAudio{
			Data:   encodedAudio,
			Format: format,
		},
	}, nil
}

func (c *Client) createImageContentFromBinary(binary []byte) (api.ImageContent, error) {
	mime, err := getMimeTypeFromBytes(binary)
	if err != nil {
		return api.ImageContent{}, err
	}

	encoded := base64.StdEncoding.EncodeToString(binary)
	content := api.ImageContent{
		Type: imageURLType,
		ImageURL: struct {
			URL string `json:"url"`
		}{
			URL: fmt.Sprintf(imageContent, mime, encoded),
		},
	}

	return content, nil
}

func (c *Client) createImageContentFromURLOrFile(image string) (api.ImageContent, error) {
	var content api.ImageContent

	if isValidURL(image) {
		content = api.ImageContent{
			Type: imageURLType,
			ImageURL: struct {
				URL string `json:"url"`
			}{
				URL: image,
			},
		}
	} else {
		mime, err := c.getMimeTypeFromFileContent(image)
		if err != nil {
			return content, err
		}

		encodedImage, err := c.base64Encode(image)
		if err != nil {
			return content, err
		}

		content = api.ImageContent{
			Type: imageURLType,
			ImageURL: struct {
				URL string `json:"url"`
			}{
				URL: fmt.Sprintf(imageContent, mime, encodedImage),
			},
		}
	}

	return content, nil
}

func (c *Client) detectAudioFormat(path string) (string, error) {
	file, err := c.reader.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf, err := c.reader.ReadBufferFromFile(file)
	if err != nil {
		return "", err
	}

	// WAV
	if string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WAVE" {
		return "wav", nil
	}

	// MP3 (ID3 or sync bits)
	if string(buf[0:3]) == "ID3" || (buf[0] == 0xFF && (buf[1]&0xE0) == 0xE0) {
		return "mp3", nil
	}

	// FLAC
	if string(buf[0:4]) == "fLaC" {
		return "flac", nil
	}

	// OGG
	if string(buf[0:4]) == "OggS" {
		return "ogg", nil
	}

	// M4A / MP4
	if string(buf[4:8]) == "ftyp" {
		if string(buf[8:12]) == "M4A " || string(buf[8:12]) == "isom" || string(buf[8:12]) == "mp42" {
			return "m4a", nil
		}
		return "mp4", nil
	}

	return "unknown", nil
}

func (c *Client) getMimeTypeFromFileContent(path string) (string, error) {
	file, err := c.reader.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer, err := c.reader.ReadBufferFromFile(file)
	if err != nil {
		return "", err
	}

	mimeType := stdhttp.DetectContentType(buffer)

	return mimeType, nil
}

func (c *Client) postAndWriteBinaryOutput(endpoint string, requestBody interface{}, outputPath, debugLabel string, transform func([]byte) ([]byte, error)) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	c.printRequestDebugInfo(endpoint, body, nil)

	respBytes, err := c.Caller.Post(endpoint, body, false)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	if transform != nil {
		respBytes, err = transform(respBytes)
		if err != nil {
			return err
		}
	}

	outFile, err := c.writer.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := c.writer.Write(outFile, respBytes); err != nil {
		return fmt.Errorf("failed to write %s: %w", debugLabel, err)
	}

	c.printResponseDebugInfo([]byte(fmt.Sprintf("[%s] %d bytes written to %s", debugLabel, len(respBytes), outputPath)))
	return nil
}

func getExtension(path string) string {
	ext := filepath.Ext(path) // e.g. ".mp4"
	if ext != "" {
		return strings.TrimPrefix(ext, ".") // "mp4"
	}
	return ""
}

func getMimeTypeFromBytes(data []byte) (string, error) {
	mimeType := stdhttp.DetectContentType(data)

	return mimeType, nil
}

func isValidURL(input string) bool {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	// Ensure that the URL has a valid scheme
	schemes := []string{httpScheme, httpsScheme}
	for _, scheme := range schemes {
		if strings.HasPrefix(parsedURL.Scheme, scheme) {
			return true
		}
	}

	return false
}
