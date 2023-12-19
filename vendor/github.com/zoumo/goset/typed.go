/*
Copyright 2019 Jim Zhang (jim.zoumo@gmail.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package goset

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// typedSet is a set with specified type
type typedSet interface {
	// Add adds an element to the set anyway.
	Add(elem ...interface{})

	// Remove removes an element from the set.
	Remove(elem ...interface{})

	// Contains checks whether the given item is in the set.
	Contains(item interface{}) bool

	// Copy clones the set.
	Copy() typedSet

	// Len returns the size of set. aka Cardinality.
	Len() int

	// Range calls f sequentially for each element present in the set.
	// If f returns false, range stops the iteration.
	//
	// Note: the iteration order is not specified and is not guaranteed
	// to be the same from one iteration to the next. The index only
	// means how many elements have been visited in the iteration, it not
	// specifies the index of an element in the set
	Range(foreach func(index int, elem interface{}) bool)
	// // ContainsAny returns true if any elems are contained in the set
	// ContainsAny(elems ...interface{}) bool

	// Equal checks whether this set is equal to the given one.
	// There are two constraints if set a is equal to set b.
	// the two set must have the same size and contain the same elements.
	Equal(b typedSet) bool

	// IsSubsetOf checks whether this set is the subset of the given set
	// In other words, all elements in this set are also the elements
	// of the given set.
	IsSubsetOf(b typedSet) bool

	// ---------------------------------------------------------------------
	// Set Oprations

	// Diff returns the difference between the set and this given
	// one, aka Difference Set
	// math formula: a - b
	Diff(b typedSet) typedSet

	// SymmetricDiff returns the symmetric difference between this set
	// and the given one. aka Symmetric Difference Set
	// math formula: (a - b) ∪ (b - a)
	SymmetricDiff(b typedSet) typedSet

	// Unite combines two sets into a new one, aka Union Set
	// math formula: a ∪ b
	Unite(b typedSet) typedSet

	// Intersect returns the intersection of two set, aka Intersection Set
	// math formula: a ∩ b
	Intersect(b typedSet) typedSet
}

type typed int

const (
	typedInt typed = iota
	typedString
	typedAny
)

var (
	allTyped = []typed{
		typedInt,
		typedString,
		typedAny,
	}
)

func typedAssert(elem interface{}) typed {
	switch elem.(type) {
	case int:
		return typedInt
	case string:
		return typedString
	}
	return typedAny
}

func concurrent(f func(typed)) {
	wg := sync.WaitGroup{}
	for _, t := range allTyped {
		wg.Add(1)
		go func(t typed) {
			defer wg.Done()
			f(t)
		}(t)
	}
	wg.Wait()
}

func synchronous(f func(typed)) {
	for _, t := range allTyped {
		f(t)
	}
}

var visitAll = synchronous

type typedSetGroup map[typed]typedSet

func newTypedSetGroup(elems ...interface{}) typedSetGroup {
	s := make(typedSetGroup)
	s.store(typedInt, newInts())
	s.store(typedString, newStrings())
	s.store(typedAny, newAny())

	s.Add(elems...)
	return s
}

func (s typedSetGroup) store(t typed, in typedSet) {
	s[t] = in
}

func (s typedSetGroup) load(t typed) typedSet {
	v, _ := s[t]
	return v.(typedSet)
}

func (s typedSetGroup) typedSetFor(elem interface{}) typedSet {
	v := s.load(typedAssert(elem))
	return v.(typedSet)
}

func (s typedSetGroup) Add(elems ...interface{}) (err error) {
	defer func() {
		// recover unhashable error
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	for _, elem := range elems {
		s.typedSetFor(elem).Add(elem)
	}
	return nil
}

func (s typedSetGroup) Remove(elems ...interface{}) {
	for _, elem := range elems {
		s.typedSetFor(elem).Remove(elem)
	}
}

func (s typedSetGroup) Range(foreach func(index int, elem interface{}) bool) {
	var i int64 = -1
	visitAll(func(t typed) {
		s.load(t).Range(func(_ int, elem interface{}) bool {
			// add firstly to get the right index
			newi := atomic.AddInt64(&i, 1)
			return foreach(int(newi), elem)
		})
	})
}

func (s typedSetGroup) Contains(elem interface{}) (ret bool) {
	defer func() {
		// recover unhashable error
		if e := recover(); e != nil {
			ret = false
			return
		}
	}()

	return s.typedSetFor(elem).Contains(elem)
}

func (s typedSetGroup) ContainsAll(elems ...interface{}) bool {
	for _, elem := range elems {
		if !s.Contains(elem) {
			return false
		}
	}
	return true
}

func (s typedSetGroup) ContainsAny(elems ...interface{}) bool {
	for _, elem := range elems {
		if s.Contains(elem) {
			return true
		}
	}
	return false
}

func (s typedSetGroup) Copy() typedSetGroup {
	ret := make(typedSetGroup)
	visitAll(func(t typed) {
		ret.store(t, s.load(t).Copy())
	})
	return ret
}

func (s typedSetGroup) Len() int {
	var l int64
	visitAll(func(t typed) {
		atomic.AddInt64(&l, int64(s.load(t).Len()))
	})
	return int(l)
}

func (s typedSetGroup) Equal(b typedSetGroup) bool {
	var fail int64
	visitAll(func(t typed) {
		if !s.load(t).Equal(b.load(t)) {
			atomic.AddInt64(&fail, 1)
		}
	})
	return fail == 0
}

func (s typedSetGroup) IsSubsetOf(b typedSetGroup) bool {
	var fail int64
	visitAll(func(t typed) {
		if !s.load(t).IsSubsetOf(b.load(t)) {
			atomic.AddInt64(&fail, 1)
		}
	})
	return fail == 0
}

func (s typedSetGroup) Diff(b typedSetGroup) typedSetGroup {
	ret := make(typedSetGroup)
	visitAll(func(t typed) {
		ret.store(t, s.load(t).Diff(b.load(t)))
	})
	return ret
}

func (s typedSetGroup) SymmetricDiff(b typedSetGroup) typedSetGroup {
	ret := make(typedSetGroup)
	visitAll(func(t typed) {
		ret.store(t, s.load(t).SymmetricDiff(b.load(t)))
	})
	return ret
}

func (s typedSetGroup) Unite(b typedSetGroup) typedSetGroup {
	ret := make(typedSetGroup)
	visitAll(func(t typed) {
		ret.store(t, s.load(t).Unite(b.load(t)))
	})
	return ret
}

func (s typedSetGroup) Intersect(b typedSetGroup) typedSetGroup {
	ret := make(typedSetGroup)
	visitAll(func(t typed) {
		ret.store(t, s.load(t).Intersect(b.load(t)))
	})
	return ret
}

func (s typedSetGroup) Elements() []interface{} {
	ret := make([]interface{}, 0, s.Len())

	// synchronous to avoid write race
	synchronous(func(t typed) {
		s.load(t).Range(func(_ int, elem interface{}) bool {
			ret = append(ret, elem)
			return true
		})
	})
	return ret
}
