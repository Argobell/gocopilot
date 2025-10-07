package logger

type NoopLogger struct{}

func (n NoopLogger) Debug(format string, args ...interface{}) {}
func (n NoopLogger) Info(format string, args ...interface{})  {}
func (n NoopLogger) Warn(format string, args ...interface{})  {}
func (n NoopLogger) Error(format string, args ...interface{}) {}