package clru

import (
	// "fmt"
	"os"
	"sync"
	"testing"
	"time"
)

type TestStruct struct {
	N int
	S string
}

func Inc(entry *Entry) {
	entry.Value = entry.Value.(int) + 1
}

func TestGet(t *testing.T) {
	c := New(32, NoExpiration)
	c.Add("key", 123)
	t.Log(c.Len())
	if v, ok := c.Get("key"); !ok || v.(int) != 123 {
		t.Fatalf("expected get to return 123 but got %v", v)
	}

	if _, ok := c.Get("key2"); ok {
		t.Fatal("it shouldn't exist")
	}

	s := TestStruct{1, "one"}
	c.Add("key", &s)

	r, ok := c.Get("key")
	if !ok {
		t.Fatal("data is lost")
	}

	rs := r.(*TestStruct)
	if rs.N != 1 || rs.S != "one" {
		t.Fatal("wtf!")
	}
}

func TestEvict(t *testing.T) {
	evicted := 0
	onEvicted := func(e *Entry) {
		t.Logf("on evicted %d", e.Value)
		evicted = e.Value.(int)
	}
	c := New(1, NoExpiration)
	c.OnEvicted = onEvicted
	c.Add("key", 1)
	c.Evict("key")

	if evicted != 1 {
		t.Fatal("expected pop 1")
	}
	if _, ok := c.Get("key"); ok {
		t.Fatal("it shouldn't exist")
	}
}

func TestUpdate(t *testing.T) {
	c := New(1, NoExpiration)
	c.Add("key", 1)
	if e, ok := c.Update("key", Inc); !ok || e.Value.(int) != 2 {
		t.Fatal("it should be 2")
	}
}

func TestTTL(t *testing.T) {
	c := New(1, 100*time.Millisecond)
	c.Add("key", 1)
	time.Sleep(100 * time.Millisecond)
	if _, ok := c.Get("key"); ok {
		t.Fatal("expiration not work")
	}
}

func TestSerialize(t *testing.T) {
	c := New(1, NoExpiration)

	c.Add("key1", 1)
	c.Add("key2", &TestStruct{1, "one"})

	fname := os.TempDir() + "/clru.test.data"

	if err := c.SaveFile(fname); err != nil {
		t.Fatal("save failed", err)
	}

	var err error
	if c, err = NewWithFile(1, NoExpiration, fname); err != nil {
		t.Fatal("load failed", err)
	}

	if v, ok := c.Get("key1"); !ok || v.(int) != 1 {
		t.Fatal("expected get to return 1")
	}

	r, _ := c.Get("key2")
	rs := r.(*TestStruct)
	if rs.N != 1 || rs.S != "one" {
		t.Fatal("wtf!")
	}
}

func BenchmarkAdd(b *testing.B) {
	var wg sync.WaitGroup
	c := New(1000, NoExpiration)

	ch := make(chan int, 10000)

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			for i := range ch {
				c.Add(Key(i), i)
			}
			wg.Done()
		}()
	}

	for i := 0; i < b.N; i++ {
		ch <- i
	}
	close(ch)
	wg.Wait()
}
