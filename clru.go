package clru

import (
	// "bytes"
	"container/list"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"os"
	"time"
)

const NUM_SHARD int = 32
const NoExpiration time.Duration = 0

type CLRU struct {
	MaxEntries int
	TTL        time.Duration
	OnEvicted  Callback

	maxEPS int
	shards []*LRUShard
}

func New(maxEntries int, ttl time.Duration) *CLRU {
	maxEPS := int(math.Ceil(float64(maxEntries) / float64(NUM_SHARD)))

	shards := make([]*LRUShard, NUM_SHARD, NUM_SHARD)
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
	dec := gob.NewDecoder(r)
	for {
		var e Entry
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				return c, nil
			}
			return nil, err
		} else {
			c.getShard(e.Key).PutIfAbsent(&e)
		}
	}
}

func NewWithFile(maxEntries int, ttl time.Duration, filename string) (c *CLRU, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return
	}
	return NewWithReader(maxEntries, ttl, f)
}

func (c *CLRU) getShard(key Key) *LRUShard {
	// var buf bytes.Buffer
	// enc := gob.NewEncoder(&buf)
	// err := enc.Encode(key)
	// if err != nil {
	// panic("the key cannot convert to []byte")
	// }
	// b := buf.Bytes()
	b := []byte(key)

	hasher := fnv.New32()
	hasher.Write(b)
	return c.shards[uint(hasher.Sum32())%uint(NUM_SHARD)]
}

func (c *CLRU) getEntry(shard *LRUShard, key Key) (entry *Entry, found bool) {
	var el *list.Element
	if el, found = shard.Get(key); !found {
		return
	}
	entry = el.Value.(*Entry)
	now := time.Now()
	if c.TTL != NoExpiration && now.Sub(entry.ATime) > c.TTL {
		c.removeElement(shard, el)
		return entry, false
	}
	entry.ATime = now
	shard.Offer(el)
	return entry, true
}

func (c *CLRU) removeElement(shard *LRUShard, el *list.Element) {
	shard.Remove(el)
	if c.OnEvicted != nil {
		//dead lock?
		c.OnEvicted(el.Value.(*Entry))
	}
}

func (c *CLRU) Add(key Key, value interface{}) {
	shard := c.getShard(key)
	shard.Lock()

	if entry, found := c.getEntry(shard, key); found {
		entry.Value = value
	} else {
		now := time.Now()
		entry = &Entry{Key: key, Value: value, ATime: now, CTime: now}
		shard.PutIfAbsent(entry)
		for shard.Len() > c.maxEPS {
			c.removeElement(shard, shard.Oldest())
		}
	}
	shard.Unlock()
}

func (c *CLRU) Get(key Key) (value interface{}, found bool) {
	shard := c.getShard(key)
	shard.Lock()
	if entry, found := c.getEntry(shard, key); found {
		shard.Unlock()
		return entry.Value, true
	}
	shard.Unlock()
	return
}

func (c *CLRU) GetEntry(key Key) (entry *Entry, found bool) {
	shard := c.getShard(key)
	shard.Lock()
	defer shard.Unlock()
	return c.getEntry(shard, key)
}

func (c *CLRU) Update(key Key, op Callback) (entry *Entry, found bool) {
	shard := c.getShard(key)
	shard.Lock()

	if entry, found = c.getEntry(shard, key); found {
		op(entry)
	}
	shard.Unlock()
	return
}

func (c *CLRU) Evict(key Key) {
	shard := c.getShard(key)
	shard.Lock()

	if el, found := shard.Get(key); !found {
		shard.Unlock()
		return
	} else {
		c.removeElement(shard, el)
		shard.Unlock()
	}
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
