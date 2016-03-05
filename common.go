package clru

import (
	"time"
)

type Key string
type Value interface{}

type Entry struct {
	Key   Key
	Value Value
	CTime time.Time // create time
	ATime time.Time // attach time
}

type Callback func(entry *Entry)
