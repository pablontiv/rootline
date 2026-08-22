package query

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type ExprCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type cacheEntry struct {
	key     string
	program *vm.Program
}

func NewExprCache(capacity int) *ExprCache {
	return &ExprCache{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

func (c *ExprCache) Compile(whereExpr string) (*vm.Program, error) {
	c.mu.RLock()
	if elem, ok := c.items[whereExpr]; ok {
		c.mu.RUnlock()
		c.mu.Lock()
		c.order.MoveToFront(elem)
		c.mu.Unlock()
		return elem.Value.(*cacheEntry).program, nil
	}
	c.mu.RUnlock()

	// Disable the expr-lang builtin type() function so the record's "type"
	// field can be used in where expressions (issue #139). The type() builtin
	// would otherwise shadow the field access and cause a type mismatch error.
	program, err := expr.Compile(whereExpr, expr.AsBool(), expr.AllowUndefinedVariables(), expr.DisableBuiltin("type"))
	if err != nil {
		return nil, fmt.Errorf("compiling where expression: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[whereExpr]; ok {
		c.order.MoveToFront(elem)
		return elem.Value.(*cacheEntry).program, nil
	}

	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}

	entry := &cacheEntry{key: whereExpr, program: program}
	elem := c.order.PushFront(entry)
	c.items[whereExpr] = elem

	return program, nil
}
