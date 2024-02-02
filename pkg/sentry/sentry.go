package sentry

import (
	"fmt"
	"os"
	"time"

	sentrygo "github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
)

var FlushTime = time.Second * 5

// Sentry hold information for a sentry scope
type Sentry struct {
	context       echo.Context
	error         error
	message       string
	level         sentrygo.Level
	extras        map[string]interface{}
	tags          map[string]string
	contextValues map[string]sentrygo.Context
}

func (s *Sentry) WithContext(c echo.Context) *Sentry {
	s.context = c
	return s
}

func (s *Sentry) WithError(err error) *Sentry {
	s.error = err
	return s
}

func (s *Sentry) WithMessage(message string) *Sentry {
	s.message = message
	return s
}

func (s *Sentry) WithLevel(level sentrygo.Level) *Sentry {
	s.level = level
	return s
}

func (s *Sentry) WithExtras(extras map[string]interface{}) *Sentry {
	s.extras = extras
	return s
}

func (s *Sentry) WithTags(tags map[string]string) *Sentry {
	s.tags = tags
	return s
}

func (s *Sentry) WithContextValues(ctx map[string]sentrygo.Context) *Sentry {
	s.contextValues = ctx
	return s
}

// configScope configure all information into current scope
func (s *Sentry) configScope(scope *sentrygo.Scope) {
	// set level
	scope.SetLevel(s.level)
	// set extras
	scope.SetExtras(s.extras)
	// set tags
	scope.SetTags(s.tags)
	// set context values
	scope.SetContexts(s.contextValues)
}

func (s *Sentry) getHub() *sentrygo.Hub {
	currentHub := sentrygo.CurrentHub().Clone()
	if s.context != nil {
		hub := sentryecho.GetHubFromContext(s.context)
		if hub != nil {
			currentHub = hub
		}
	}
	return currentHub
}

func (s *Sentry) sendError() {
	if os.Getenv("APP_ENV") == "local" || len(os.Getenv("SENTRY_DSN")) == 0 {
		return
	}

	hub := s.getHub()

	// config basic info into scope
	hub.ConfigureScope(s.configScope)
	// capture error and send
	hub.CaptureException(s.error)
	// clear context data
	hub.ConfigureScope(func(scope *sentrygo.Scope) {
		scope.Clear()
	})
}

func (s *Sentry) sendMessage() {
	if os.Getenv("APP_ENV") == "local" || len(os.Getenv("SENTRY_DSN")) == 0 {
		return
	}

	hub := s.getHub()

	// config basic info into scope
	hub.ConfigureScope(s.configScope)
	// capture message and send
	hub.CaptureMessage(s.message)
	// clear context data
	hub.ConfigureScope(func(scope *sentrygo.Scope) {
		scope.Clear()
	})
}

func (s *Sentry) Debug(message string) {
	s.WithMessage(message).WithLevel(sentrygo.LevelDebug).sendMessage()
}

func (s *Sentry) Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.WithMessage(msg).WithLevel(sentrygo.LevelDebug).sendMessage()
}

func (s *Sentry) Info(message string) {
	s.WithMessage(message).WithLevel(sentrygo.LevelInfo).sendMessage()
}

func (s *Sentry) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.WithMessage(msg).WithLevel(sentrygo.LevelInfo).sendMessage()
}

func (s *Sentry) Warning(message string) {
	s.WithMessage(message).WithLevel(sentrygo.LevelWarning).sendMessage()
}

func (s *Sentry) Warningf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.WithMessage(msg).WithLevel(sentrygo.LevelWarning).sendMessage()
}

func (s *Sentry) Error(err error) {
	s.WithError(err).WithLevel(sentrygo.LevelError).sendError()
}

func (s *Sentry) Errorf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	s.WithError(err).WithLevel(sentrygo.LevelError).sendError()
}

func (s *Sentry) Fatal(err error) {
	s.WithError(err).WithLevel(sentrygo.LevelFatal).sendError()
	sentrygo.Flush(FlushTime)
}

func (s *Sentry) Fatalf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	s.WithError(err).WithLevel(sentrygo.LevelFatal).sendError()
	sentrygo.Flush(FlushTime)
}

// Convenient functions. Use directly, no need to create sentry instance
func createSentry() *Sentry {
	return new(Sentry)
}

// WithContext set context into the current scope
func WithContext(c echo.Context) *Sentry {
	sentry := createSentry()
	sentry.WithContext(c)
	return sentry
}

// WithExtras set extras infor into current scope
func WithExtras(extras map[string]interface{}) *Sentry {
	sentry := createSentry()
	sentry.WithExtras(extras)
	return sentry
}

// WithTags set extras infor into current scope
func WithTags(tags map[string]string) *Sentry {
	sentry := createSentry()
	sentry.WithTags(tags)
	return sentry
}

// WithContextValues set extras infor into current scope
func WithContextValues(contextValues map[string]sentrygo.Context) *Sentry {
	sentry := createSentry()
	sentry.WithContextValues(contextValues)
	return sentry
}

// Debug send debug information to sentry server
func Debug(message string) {
	createSentry().WithMessage(message).WithLevel(sentrygo.LevelDebug).sendMessage()
}

// Debugf send debug formatted message to sentry server
func Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	createSentry().WithMessage(msg).WithLevel(sentrygo.LevelDebug).sendMessage()
}

// Info send information to sentry server
func Info(message string) {
	createSentry().WithMessage(message).WithLevel(sentrygo.LevelInfo).sendMessage()
}

// Infof send formatted message to sentry server
func Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	createSentry().WithMessage(msg).WithLevel(sentrygo.LevelInfo).sendMessage()
}

// Warning send warning message to sentry server
func Warning(message string) {
	createSentry().WithMessage(message).WithLevel(sentrygo.LevelWarning).sendMessage()
}

// Warningf send formatted message to sentry server
func Warningf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	createSentry().WithMessage(msg).WithLevel(sentrygo.LevelWarning).sendMessage()
}

// Error send error to sentry server
func Error(err error) {
	createSentry().WithError(err).WithLevel(sentrygo.LevelError).sendError()
}

// Errorf send error with custom message to sentry server
func Errorf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	createSentry().WithError(err).WithLevel(sentrygo.LevelError).sendError()
}

// Fatal send fatal signal to sentry server
func Fatal(err error) {
	createSentry().WithError(err).WithLevel(sentrygo.LevelFatal).sendError()
	sentrygo.Flush(FlushTime)
}

// Fatalf send fatal signal with custom message to sentry server
func Fatalf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	createSentry().WithError(err).WithLevel(sentrygo.LevelFatal).sendError()
	sentrygo.Flush(FlushTime)
}
