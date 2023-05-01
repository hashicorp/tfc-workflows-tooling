// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sethvargo/go-retry"
)

// default to 1 hour, allow override
const (
	defaultTimeoutDuration = 1 * time.Hour
	tfMaxTimeout           = "TF_MAX_TIMEOUT"
)

var (
	once = new(sync.Once)
)

type RetryTimeoutError struct {
	msg string
}

func newRetryTimeoutError(operation string) *RetryTimeoutError {
	return &RetryTimeoutError{
		msg: fmt.Sprintf("%s has exceeded maximum timeout", operation),
	}
}

func retryableTimeoutError(operation string) error {
	return retry.RetryableError(newRetryTimeoutError(operation))
}

func (retryErr *RetryTimeoutError) Error() string { return retryErr.msg }

func defaultBackoff() retry.Backoff {
	backoff := retry.NewFibonacci(2 * time.Second)
	backoff = retry.WithCappedDuration(7*time.Second, backoff)
	backoff = retry.WithMaxDuration(Timeout(), backoff)
	return backoff
}

func Timeout() time.Duration {
	timeout := defaultTimeoutDuration
	once.Do(func() {
		timeoutEnv := os.Getenv(tfMaxTimeout)
		if timeoutEnv == "" {
			return
		}

		t, err := time.ParseDuration(timeoutEnv)
		if err != nil {
			log.Printf("[ERROR] issue setting timeout duration with %s", err.Error())
			return
		}

		log.Printf("[DEBUG] timeout duration has successfully been set as %v", t)
		timeout = t
	})
	return timeout
}
