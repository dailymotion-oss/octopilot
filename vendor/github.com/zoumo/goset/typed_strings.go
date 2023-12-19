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

type strings map[string]Empty

func newStrings(elems ...string) strings {
	s := make(strings)
	for _, elem := range elems {
		s.Add(elem)
	}
	return s
}

func (s strings) Len() int {
	return len(s)
}

func (s strings) Add(items ...interface{}) {
	for _, item := range items {
		s[item.(string)] = Empty{}
	}
}

func (s strings) Remove(items ...interface{}) {
	for _, item := range items {
		delete(s, item.(string))
	}
}

func (s strings) Contains(item interface{}) bool {
	_, ok := s[item.(string)]
	return ok
}

func (s strings) Equal(b typedSet) bool {
	if s.Len() == b.Len() {
		return s.isSubsetOf(b)
	}
	return false
}

func (s strings) IsSubsetOf(b typedSet) bool {
	if s.Len() > b.Len() {
		return false
	}
	return s.isSubsetOf(b)
}

func (s strings) isSubsetOf(b typedSet) bool {
	for key := range s {
		if !b.Contains(key) {
			return false
		}
	}
	return true
}

func (s strings) Copy() typedSet {
	copy := make(strings, s.Len())
	for key := range s {
		copy[key] = Empty{}
	}
	return copy
}

func (s strings) Diff(b typedSet) typedSet {
	s2 := b.(strings)
	diff := newStrings()
	for key := range s {
		if !s2.Contains(key) {
			diff.Add(key)
		}
	}
	return diff
}

func (s strings) SymmetricDiff(b typedSet) typedSet {
	s2 := b.(strings)
	adiff := s.Diff(s2)
	bdiff := s2.Diff(s)
	return adiff.Unite(bdiff)
}

func (s strings) Unite(b typedSet) typedSet {
	s2 := b.(strings)
	union := s.Copy()
	for key := range s2 {
		union.Add(key)
	}
	return union
}

func (s strings) Intersect(b typedSet) typedSet {
	s2 := b.(strings)

	var x, y strings
	// find the smaller one
	if s.Len() <= s2.Len() {
		x = s
		y = s2
	} else {
		x = s2
		y = s
	}

	intersection := newStrings()
	for key := range x {
		if y.Contains(key) {
			intersection.Add(key)
		}
	}
	return intersection
}

func (s strings) Range(foreach func(i int, elem interface{}) bool) {
	i := 0
	for key := range s {
		if !foreach(i, key) {
			break
		}
		i++
	}
}

func (s strings) List() []string {
	res := make([]string, 0, len(s))
	for i := range s {
		res = append(res, i)
	}
	sort.Strings(res)
	return res
}
