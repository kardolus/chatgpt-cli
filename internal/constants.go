package internal

type contextKey string

const (
	BinaryDataKey          contextKey = "binaryData"
	ImagePathKey           contextKey = "imagePath"
	AudioPathKey           contextKey = "audioPath"
	HeaderContentTypeKey              = "Content-Type"
	HeaderContentTypeValue            = "application/json"
	HeaderUserAgentKey                = "User-Agent"
	HeaderAuthorizationKey            = "Authorization"
)
