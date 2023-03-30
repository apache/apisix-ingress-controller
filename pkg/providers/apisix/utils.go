package apisix

import "time"

func GetSyncDelay(interval time.Duration, count int) time.Duration {
	if count <= 0 {
		return 0
	}
	delay := time.Duration(int64(interval) / int64(count))
	if delay < time.Millisecond {
		delay = 0
	}
	return delay
}
