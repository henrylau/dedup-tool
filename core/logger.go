package core

// Logger interface for logging messages.
type Logger interface {
	Info(message string)
	Error(message string)
}
