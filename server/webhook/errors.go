package webhook

import "errors"

var (
	ErrInvalidPayloadType    = errors.New("invalid payload type for handler")
	ErrNoFilePath            = errors.New("no file path in event")
	ErrNoFilesInEvent        = errors.New("no files in event")
	ErrEventTypeNotSupported = errors.New("event type not supported for processing")
	ErrHandlerNotFound       = errors.New("no handler found for source and event type")
	ErrWebhooksNotConfigured = errors.New("webhooks not configured")
	ErrMissingAPIKey         = errors.New("missing API key")
	ErrInvalidAPIKey         = errors.New("invalid API key")
	ErrMissingSource         = errors.New("source query parameter is required")
	ErrFailedToReadBody      = errors.New("failed to read request body")
	ErrFailedToParse         = errors.New("failed to parse webhook payload")
	ErrFailedToProcess       = errors.New("failed to process webhook")
)

type WebhookError struct {
	Source  string
	Event   string
	Message string
	Err     error
}

func (e *WebhookError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *WebhookError) Unwrap() error {
	return e.Err
}

func NewWebhookError(source, event, message string, err error) *WebhookError {
	return &WebhookError{
		Source:  source,
		Event:   event,
		Message: message,
		Err:     err,
	}
}
