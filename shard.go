package clru

import (
	"container/list"
	"sync"
)

type LRUShard struct {
	sync.Mutex
	lst   *list.List
	table map[Key]*list.Element
}

func newShard() *LRUShard {
	return &LRUShard{lst: list.New(), table: make(map[Key]*list.Element)}
}

func (s *LRUShard) Get(key Key) (el *list.Element, found bool) {
	el, found = s.table[key]
	return
}

func (s *LRUShard) Put(entry *Entry) {
	s.table[entry.Key] = s.lst.PushFront(entry)
}

func (s *LRUShard) PutIfAbsent(entry *Entry) {
	if _, found := s.table[entry.Key]; !found {
		s.Put(entry)
	}
}

func (s *LRUShard) Oldest() (el *list.Element) {
	return s.lst.Back()
}

func (s *LRUShard) Remove(el *list.Element) {
	entry := el.Value.(*Entry)
	delete(s.table, entry.Key)
	s.lst.Remove(el)
}

func (s *LRUShard) Offer(el *list.Element) {
	s.lst.MoveToFront(el)
}

func (s *LRUShard) Len() int {
	return s.lst.Len()
}
