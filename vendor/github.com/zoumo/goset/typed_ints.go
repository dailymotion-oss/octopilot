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

import "sort"

type ints map[int]Empty

func newInts(elems ...int) ints {
	s := make(ints)
	for _, elem := range elems {
		s.Add(elem)
	}
	return s
}

func (s ints) Len() int {
	return len(s)
}

func (s ints) Add(items ...interface{}) {
	for _, item := range items {
		s[item.(int)] = Empty{}
	}
}

func (s ints) Remove(items ...interface{}) {
	for _, item := range items {
		delete(s, item.(int))
	}
}

func (s ints) Contains(item interface{}) bool {
	_, ok := s[item.(int)]
	return ok
}

func (s ints) Equal(b typedSet) bool {
	if s.Len() == b.Len() {
		return s.isSubsetOf(b)
	}
	return false
}

func (s ints) IsSubsetOf(b typedSet) bool {
	if s.Len() > b.Len() {
		return false
	}
	return s.isSubsetOf(b)
}

func (s ints) isSubsetOf(b typedSet) bool {
	for key := range s {
		if !b.Contains(key) {
			return false
		}
	}
	return true
}

func (s ints) Copy() typedSet {
	copy := make(ints, s.Len())
	for key := range s {
		copy[key] = Empty{}
	}
	return copy
}

func (s ints) Diff(b typedSet) typedSet {
	s2 := b.(ints)
	diff := newInts()
	for key := range s {
		if !s2.Contains(key) {
			diff.Add(key)
		}
	}
	return diff
}

func (s ints) SymmetricDiff(b typedSet) typedSet {
	s2 := b.(ints)
	adiff := s.Diff(s2)
	bdiff := s2.Diff(s)
	return adiff.Unite(bdiff)
}

func (s ints) Unite(b typedSet) typedSet {
	s2 := b.(ints)
	union := s.Copy()
	for key := range s2 {
		union.Add(key)
	}
	return union
}

func (s ints) Intersect(b typedSet) typedSet {
	s2 := b.(ints)

	var x, y ints
	// find the smaller one
	if s.Len() <= s2.Len() {
		x = s
		y = s2
	} else {
		x = s2
		y = s
	}

	intersection := newInts()
	for key := range x {
		if y.Contains(key) {
			intersection.Add(key)
		}
	}
	return intersection
}

func (s ints) Range(foreach func(i int, elem interface{}) bool) {
	i := 0
	for key := range s {
		if !foreach(i, key) {
			break
		}
		i++
	}
}

func (s ints) List() []int {
	res := make([]int, 0, len(s))
	for i := range s {
		res = append(res, i)
	}
	sort.Ints(res)
	return res
}
