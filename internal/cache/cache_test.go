package cache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/Ser9unin/ImagePrev/internal/config"
	"github.com/stretchr/testify/require"
)

var cap config.CacheCfg

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		cap.Capacity = 10
		c := NewCache(cap)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		cap.Capacity = 5
		c := NewCache(cap)

		wasInCache := c.Set("aaa", 100)
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200)
		require.False(t, wasInCache)

		val, ok := c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb")
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300)
		require.True(t, wasInCache)

		val, ok = c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc")
		require.False(t, ok)
		require.Nil(t, val)
	})

	t.Run("purge logic", func(t *testing.T) {
		// Write me
	})
}

func TestCacheMultithreading(t *testing.T) {
	t.Skip() // Remove me if task with asterisk completed.
	cap.Capacity = 10

	c := NewCache(cap)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Set(strconv.Itoa(i), i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Get(strconv.Itoa(rand.Intn(1_000_000)))
		}
	}()

	wg.Wait()
}
