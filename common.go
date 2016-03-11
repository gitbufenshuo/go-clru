package clru

import (
	"time"
)

type Entry struct {
	Key   string
	Value interface{}
	CTime time.Time // create time
	ATime time.Time // attach time
}

type Callback func(entry *Entry)
