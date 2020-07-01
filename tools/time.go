package tools

import (
	"fmt"
	"time"
)

// Changes RFC3339 strings to unix timestamps
func RFC3339ToUnix(t string) int64 {
	if t == "" {
		return 0
	}
	tt, err := time.Parse(time.RFC3339, t)
	if err != nil {
		fmt.Println("Failed parsing", t, err.Error())
		return 0
	} else {
		return tt.Unix()
	}
}
