package main

import (
	"time"
)

func Now() int64 {
	return time.Now().UnixNano() / 1000000
}

func Wait(milliseconds int) {
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}
