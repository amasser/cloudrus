package cloudrus

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/sirupsen/logrus"
)

// Hook represents one hook
type Hook struct {
	service *cloudwatchlogs.CloudWatchLogs

	inputLogEvents    chan *cloudwatchlogs.InputLogEvent
	nextSequenceToken *string
	mutex             sync.Mutex
	err               chan error

	groupName  string
	streamName string
}

// NewHook creates a new hook with an AWS session taken from the environment
func NewHook(groupName string, streamName string, batchFrequency time.Duration) (hook *Hook, err error) {
	sess := session.Must(session.NewSession())

	hook, err = NewHookWithSession(groupName, streamName, batchFrequency, sess)

	return
}

// NewHookWithSession creates a new hook with a custom AWS session
func NewHookWithSession(groupName string, streamName string, batchFrequency time.Duration, sess *session.Session) (hook *Hook, err error) {
	// init the hook
	h := Hook{
		service:    cloudwatchlogs.New(sess),
		groupName:  groupName,
		streamName: streamName,
	}

	// set the log stream
	resp, err := h.service.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(h.groupName),
		LogStreamNamePrefix: aws.String(h.streamName),
	})
	if err != nil {
		return nil, err
	}

	// get the first sequence token
	if len(resp.LogStreams) > 0 {
		h.nextSequenceToken = resp.LogStreams[0].UploadSequenceToken
		return &h, nil
	}

	// if the batch frequency is above 0
	if batchFrequency > 0 {
		// maximum number of entries is 10000
		h.inputLogEvents = make(chan *cloudwatchlogs.InputLogEvent, 10000)
		// start the ticker
		go h.putBatches(time.Tick(batchFrequency))
	}

	return &h, nil
}

// Fire adds an entry
func (h *Hook) Fire(e *logrus.Entry) error {
	// use the json formatter for the entry
	json := &logrus.JSONFormatter{}
	buf, err := json.Format(e)
	if err != nil {
		return err
	}
	msg := string(buf)

	// create the event
	event := &cloudwatchlogs.InputLogEvent{
		Message:   aws.String(msg),
		Timestamp: aws.Int64(int64(e.Time.UnixNano()) / int64(time.Millisecond)),
	}

	// if batch frequency is set
	if h.inputLogEvents != nil {
		// add the event to the event channel
		h.inputLogEvents <- event
		if lastErr, ok := <-h.err; ok {
			return fmt.Errorf("%v", lastErr)
		}
	} else {
		// lame normal synchronous put

		// use a mutex to prevent race conditions
		h.mutex.Lock()
		defer h.mutex.Unlock()

		// create the event
		params := &cloudwatchlogs.PutLogEventsInput{
			LogEvents:     []*cloudwatchlogs.InputLogEvent{event},
			LogGroupName:  aws.String(h.groupName),
			LogStreamName: aws.String(h.streamName),
			SequenceToken: h.nextSequenceToken,
		}
		// put the events into the log stream
		resp, err := h.service.PutLogEvents(params)

		if err != nil {
			return err
		}

		// get the next sequence token
		h.nextSequenceToken = resp.NextSequenceToken

	}

	return nil
}

// putBatches puts the batches into the log stream synchronized by the ticker
func (h *Hook) putBatches(ticker <-chan time.Time) {
	var batch []*cloudwatchlogs.InputLogEvent
	for {
		select {
		case p := <-h.inputLogEvents:
			// add the events from the channel to the batch
			batch = append(batch, p)
		case <-ticker:
			// when ticking, create the events
			params := &cloudwatchlogs.PutLogEventsInput{
				LogEvents:     batch,
				LogGroupName:  aws.String(h.groupName),
				LogStreamName: aws.String(h.streamName),
				SequenceToken: h.nextSequenceToken,
			}
			// put the batched events into the stream
			resp, err := h.service.PutLogEvents(params)
			if err != nil {
				h.err <- err
			} else {
				// fetch the next sequence token and
				h.nextSequenceToken = resp.NextSequenceToken
				batch = nil
			}
		}
	}
}

// Levels returns all the levels
func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}
