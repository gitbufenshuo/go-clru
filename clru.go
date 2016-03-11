package clru

import (
	"container/list"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"time"
)

const NUM_SHARD int = 32
const NoExpiration time.Duration = 0

type CLRU struct {
	MaxEntries int
	TTL        time.Duration
	maxEPS     int
	shards     []*lruShard
	// will be called when evict
	OnEvicted Callback
}

func New(maxEntries int, ttl time.Duration) *CLRU {
	maxEPS := (maxEntries / NUM_SHARD) + 1
	shards := make([]*lruShard, NUM_SHARD, NUM_SHARD)
	for i := 0; i < NUM_SHARD; i++ {
		shards[i] = newShard()
	}
	return &CLRU{
		MaxEntries: maxEntries,
		TTL:        ttl,
		maxEPS:     maxEPS,
		shards:     shards,
	}
}

func NewWithReader(maxEntries int, ttl time.Duration, r io.Reader) (c *CLRU, err error) {
	c = New(maxEntries, ttl)
	err = c.Load(r)
	return
}

func NewWithFile(maxEntries int, ttl time.Duration, fname string) (c *CLRU, err error) {
	f, err := os.Open(fname)
	defer f.Close()
	if err != nil {
		return
	}
	return NewWithReader(maxEntries, ttl, f)
}

func (c *CLRU) getShard(key string) *lruShard {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	return c.shards[uint(hasher.Sum32())%uint(NUM_SHARD)]
}

func (c *CLRU) getEntry(shard *lruShard, key string) (entry *Entry, found bool) {
	var el *list.Element
	if el, found = shard.Get(key); !found {
		return
	}
	entry = el.Value.(*Entry)
	now := time.Now()
	if c.TTL != NoExpiration && now.Sub(entry.CTime) > c.TTL {
		c.removeElement(shard, el)
		return entry, false
	}
	entry.ATime = now
	shard.Offer(el)
	return
}

func (c *CLRU) removeElement(shard *lruShard, el *list.Element) {
	shard.Remove(el)
	if c.OnEvicted != nil {
		//dead lock?
		c.OnEvicted(el.Value.(*Entry))
	}
}

func (c *CLRU) Add(key string, value interface{}) {
	shard := c.getShard(key)
	shard.Lock()
	if entry, found := c.getEntry(shard, key); found {
		entry.Value = value
	} else {
		now := time.Now()
		entry = &Entry{Key: key, Value: value, ATime: now, CTime: now}
		shard.Put(entry)
		if shard.Len() > c.maxEPS {
			c.removeElement(shard, shard.Oldest())
		}
	}
	shard.Unlock()
}

func (c *CLRU) Get(key string) (value interface{}, found bool) {
	shard := c.getShard(key)
	shard.Lock()
	entry, found := c.getEntry(shard, key)
	shard.Unlock()
	if found {
		value = entry.Value
	}
	return
}

func (c *CLRU) GetEntry(key string) (entry *Entry, found bool) {
	shard := c.getShard(key)
	shard.Lock()
	entry, found = c.getEntry(shard, key)
	shard.Unlock()
	return
}

// better not to lock in the callback
func (c *CLRU) Update(key string, op Callback) (entry *Entry, found bool) {
	shard := c.getShard(key)
	shard.Lock()
	if entry, found = c.getEntry(shard, key); found {
		op(entry)
	}
	shard.Unlock()
	return
}

func (c *CLRU) Evict(key string) {
	shard := c.getShard(key)
	shard.Lock()
	if el, found := shard.Get(key); found {
		c.removeElement(shard, el)
	}
	shard.Unlock()
}

func (c *CLRU) Len() int {
	len := 0
	for i := 0; i < NUM_SHARD; i++ {
		shard := c.shards[i]
		shard.Lock()
		len += shard.Len()
		shard.Unlock()
	}
	return len
}

func (c *CLRU) Iter() <-chan *Entry {
	ch := make(chan *Entry)
	go func() {
		for i := 0; i < NUM_SHARD; i++ {
			shard := c.shards[i]
			shard.Lock()
			for el := shard.Oldest(); el != nil; el = el.Prev() {
				ch <- el.Value.(*Entry)
			}
			shard.Unlock()
		}
		close(ch)
	}()
	return ch
}

func (c *CLRU) Flush() {
	for i := 0; i < NUM_SHARD; i++ {
		shard := c.shards[i]
		shard.Lock()
		shard.lst = list.New()
		shard.table = make(map[string]*list.Element)
		shard.Unlock()
	}
}

func (c *CLRU) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)
	for {
		var entry Entry
		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		} else {
			c.getShard(entry.Key).PutIfAbsent(&entry)
		}
	}
}

func (c *CLRU) LoadFile(fname string) error {
	f, err := os.Open(fname)
	defer f.Close()
	if err != nil {
		return err
	}
	return c.Load(f)
}

func (c *CLRU) Save(w io.Writer) (err error) {
	gob.Register(Entry{})
	enc := gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Error registering item types with Gob library")
		}
	}()
	for entry := range c.Iter() {
		gob.Register(entry.Value)
		err := enc.Encode(entry)
		if err != nil {
			return err
		}
	}
	return err
}

func (c *CLRU) SaveFile(fname string) error {
	fp, err := os.Create(fname)
	if err != nil {
		return err
	}
	err = c.Save(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}
