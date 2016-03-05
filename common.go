package clru

import (
	"time"
)

type Key string
type Value interface{}

type Entry struct {
	Key   Key
	Value Value
	CTime time.Time
	ATime time.Time
}

type Callback func(entry *Entry)
