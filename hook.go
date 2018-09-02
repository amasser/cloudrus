package cloudrus

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// Hook represents one hook
type Hook struct {
	Session   *cloudwatchlogs.CloudWatchLogs
	LogGroup  string
	LogStream string
}

// NewHook creates a new hook with an AWS session taken from the environment
func NewHook(logGroup string, logStream string) (hook *Hook, err error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	hook.LogGroup = logGroup
	hook.LogStream = logStream
	hook, err = NewHookWithSession(logGroup, logStream, sess)

	return
}

// NewHookWithSession creates a new hook with a custom AWS session
func NewHookWithSession(logGroup string, logStream string, sess *session.Session) (hook *Hook, err error) {
	hook.LogGroup = logGroup
	hook.LogStream = logStream
	hook.Session = cloudwatchlogs.New(sess)

	return
}
