# go-clru
golang concurent lru ttl cache.

### Installation

`go get github.com/khowarizmi/go-clru`

### Document
[doc](https://godoc.org/github.com/khowarizmi/go-clru)


### Usage

```
package main

import (
	"fmt"
	"github.com/khowarizmi/go-clru"
)

func inc(entry *clru.Entry) {
	entry.Value = entry.Value.(int) + 1
}

func callback(entry *clru.Entry) {
	fmt.Println("Evict", entry.Key, entry.Value.(int))
}

func main() {
	// create a clru with 960 entries
	cache := clru.New(960, clru.NoExpiration)

	// Add key1 with int 1 to the cache
	cache.Add("key1", 1)

	// Add key2 with string "value" to the cache
	cache.Add("key2", "value")

	// Get the key1 from cache and convert to int
	key1, found := cache.Get("key1")
	if found {
		fmt.Println(key1.(int))
	}

	// Update key1 with func inc
	cache.Update("key1", inc)

	// Set evict callback
	cache.OnEvicted = callback
	cache.Evict("key1")
}
```

### Test

`go test .`

###BenchMark

`go test -bench  . -benchtime 30s`