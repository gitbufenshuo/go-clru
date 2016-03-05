package clru

import (
	"time"
)

type Key interface{}

type Entry struct {
	Key   Key
	Value interface{}
	CTime time.Time
	ATime time.Time
}

type Callback func(entry *Entry)
