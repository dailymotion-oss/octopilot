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

type any map[interface{}]Empty

func newAny(elems ...interface{}) any {
	s := make(any)
	for _, elem := range elems {
		s.Add(elem)
	}
	return s
}

func (s any) Len() int {
	return len(s)
}

func (s any) Add(items ...interface{}) {
	for _, item := range items {
		s[item] = Empty{}
	}
}

func (s any) Remove(items ...interface{}) {
	for _, item := range items {
		delete(s, item)
	}
}

func (s any) Contains(item interface{}) bool {
	_, ok := s[item]
	return ok
}

func (s any) Equal(b typedSet) bool {
	if s.Len() == b.Len() {
		return s.isSubsetOf(b)
	}
	return false
}

func (s any) IsSubsetOf(b typedSet) bool {
	if s.Len() > b.Len() {
		return false
	}
	return s.isSubsetOf(b)
}

func (s any) isSubsetOf(b typedSet) bool {
	for key := range s {
		if !b.Contains(key) {
			return false
		}
	}
	return true
}

func (s any) Copy() typedSet {
	copy := make(any, s.Len())
	for key := range s {
		copy[key] = Empty{}
	}
	return copy
}

func (s any) Diff(b typedSet) typedSet {
	s2 := b.(any)
	diff := newAny()
	for key := range s {
		if !s2.Contains(key) {
			diff.Add(key)
		}
	}
	return diff
}

func (s any) SymmetricDiff(b typedSet) typedSet {
	s2 := b.(any)
	adiff := s.Diff(s2)
	bdiff := s2.Diff(s)
	return adiff.Unite(bdiff)
}

func (s any) Unite(b typedSet) typedSet {
	s2 := b.(any)
	union := s.Copy()
	for key := range s2 {
		union.Add(key)
	}
	return union
}

func (s any) Intersect(b typedSet) typedSet {
	s2 := b.(any)

	var x, y any
	// find the smaller one
	if s.Len() <= s2.Len() {
		x = s
		y = s2
	} else {
		x = s2
		y = s
	}

	intersection := newAny()
	for key := range x {
		if y.Contains(key) {
			intersection.Add(key)
		}
	}
	return intersection
}

func (s any) Range(foreach func(i int, elem interface{}) bool) {
	i := 0
	for key := range s {
		if !foreach(i, key) {
			break
		}
		i++
	}
}

func (s any) List() []interface{} {
	res := make([]interface{}, 0, len(s))
	for i := range s {
		res = append(res, i)
	}
	return res
}
