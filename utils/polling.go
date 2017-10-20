package utils

import (
	"fmt"
	"time"
)

// Poll polls the given function until it returns true to indicate its complete or an error
func Poll(pollPeriod time.Duration, timeout time.Duration, fn func() (bool, error), message string) error {
	timeoutAt := time.Now().Add(timeout)
	for {
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		if time.Now().After(timeoutAt) {
			return fmt.Errorf("Timed out waiting for %s waited for %s", message, timeout.String())
		}
		time.Sleep(pollPeriod)
	}
}
