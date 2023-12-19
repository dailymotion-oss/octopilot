/*
Copyright 2017 Jim Zhang (jim.zoumo@gmail.com)

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
	"sync"
)

type threadSafeSet struct {
	unsafe *set
	mu     sync.RWMutex
}

func newThreadSafeSet(elems ...interface{}) *threadSafeSet {
	s := &threadSafeSet{
		unsafe: newSet(),
	}
	err := s.Add(elems...)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *threadSafeSet) Add(elems ...interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.unsafe.Add(elems...)
}

func (s *threadSafeSet) Extend(b interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.unsafe.Extend(b)
}

func (s *threadSafeSet) Remove(elems ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unsafe.Remove(elems...)

}

func (s *threadSafeSet) Copy() Set {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &threadSafeSet{
		unsafe: (s.unsafe.Copy()).(*set),
	}

}

func (s *threadSafeSet) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.Len()
}

func (s *threadSafeSet) Elements() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unsafe.Elements()
}

func (s *threadSafeSet) Contains(elem interface{}) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unsafe.Contains(elem)
}

func (s *threadSafeSet) ContainsAll(elems ...interface{}) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unsafe.ContainsAll(elems...)
}

func (s *threadSafeSet) ContainsAny(elems ...interface{}) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unsafe.ContainsAny(elems...)

}

func (s *threadSafeSet) Equal(b Set) bool {

	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.Equal(b)
}

func (s *threadSafeSet) IsSubsetOf(b Set) bool {
	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.IsSubsetOf(b)

}

func (s *threadSafeSet) IsSupersetOf(b Set) bool {
	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.IsSupersetOf(b)

}

func (s *threadSafeSet) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.String()
}

func (s *threadSafeSet) ToThreadUnsafe() Set {
	return s.unsafe
}

func (s *threadSafeSet) ToThreadSafe() Set {
	return s

}

func (s *threadSafeSet) Diff(b Set) Set {
	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.Diff(b)
}

func (s *threadSafeSet) SymmetricDiff(b Set) Set {
	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.SymmetricDiff(b)
}

func (s *threadSafeSet) Unite(b Set) Set {
	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.Unite(b)
}

func (s *threadSafeSet) Intersect(b Set) Set {
	safeb, ok := b.(*threadSafeSet)
	if ok {
		safeb.mu.RLock()
		defer safeb.mu.RUnlock()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.unsafe.Intersect(b)
}

func (s *threadSafeSet) Range(foreach func(int, interface{}) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.unsafe.Range(foreach)
}
