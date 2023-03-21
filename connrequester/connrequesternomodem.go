// +build amd64 darwin

package connrequester

import (
	"log"
	"time"
)

type ConnectionRequester struct {
}

// NewConnectionRequester will return a ConnectionRequester.
// No connection will be requested until Start is called
func NewConnectionRequester() *ConnectionRequester {
	log.Printf("running amd64 version of connection requester")
	return &ConnectionRequester{}
}

// WaitUntilUpLoop will wait until a connection has been made returning an error
// if no connection is made.
// timeout is the time given to make a connection each try.
// retryAfter is the duration between attempts, it will multipy by 1.5 after
// each try with a maximum of 2 hours.
// maxRetries is how many times it will try to make a connection. If -1 it will
// try until a connection is made.
func (cr *ConnectionRequester) WaitUntilUpLoop(
	timeout time.Duration,
	retryAfter time.Duration,
	maxRetries int) error {
		return nil
	}

// Start will start requesting for a connection to be made.
func (cr *ConnectionRequester) Start() {}

// Stop will stop requesting for a connection to be made.
func (cr *ConnectionRequester) Stop() {}

