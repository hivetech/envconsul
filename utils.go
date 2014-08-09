package main

import (
	"time"

	"github.com/armon/consul-api"
)

func Now() int64 {
	return time.Now().UnixNano() / 1000000
}

func Wait(milliseconds int) {
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}

// This function is able to call KV listing functions and retry them.
// We want to retry if there are errors because it is safe (GET request),
// and erroring early is MUCH more costly than retrying over time and
// delaying the configuration propagation.
func retryableList(f func() (consulapi.KVPairs, *consulapi.QueryMeta, error)) (consulapi.KVPairs, *consulapi.QueryMeta, error) {
	i := 0
	for {
		p, m, e := f()
		if e != nil {
			if i >= 3 {
				return nil, nil, e
			}

			i++
			// Reasonably arbitrary sleep to just try again... It is
			// a GET request so this is safe.
			time.Sleep(time.Duration(i*2) * time.Second)
		}

		return p, m, e
	}
}
