package clru

import (
	"container/list"
	"sync"
)

type lruShard struct {
	sync.Mutex
	lst   *list.List
	table map[string]*list.Element
}

func newShard() *lruShard {
	return &lruShard{lst: list.New(), table: make(map[string]*list.Element)}
}

func (s *lruShard) Get(key string) (el *list.Element, found bool) {
	el, found = s.table[key]
	return
}

func (s *lruShard) Put(entry *Entry) {
	s.table[entry.Key] = s.lst.PushFront(entry)
}

func (s *lruShard) PutIfAbsent(entry *Entry) {
	if _, found := s.table[entry.Key]; !found {
		s.Put(entry)
	}
}

func (s *lruShard) Oldest() (el *list.Element) {
	return s.lst.Back()
}

func (s *lruShard) Remove(el *list.Element) {
	entry := el.Value.(*Entry)
	delete(s.table, entry.Key)
	s.lst.Remove(el)
}

func (s *lruShard) Offer(el *list.Element) {
	s.lst.MoveToFront(el)
}

func (s *lruShard) Len() int {
	return s.lst.Len()
}
