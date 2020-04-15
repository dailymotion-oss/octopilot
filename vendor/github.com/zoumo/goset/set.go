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
	"bytes"
	"fmt"
	"reflect"
)

type set struct {
	typedSetGroup
}

func newSet(elems ...interface{}) *set {
	s := &set{newTypedSetGroup()}
	err := s.Add(elems...)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *set) Extend(b interface{}) error {
	if b == nil {
		return nil
	}

	setb, ok := b.(Set)
	if !ok {
		v := reflect.ValueOf(b)
		for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
			return fmt.Errorf("error extend set with kind: %v, only support array and slice and Set", v.Kind())
		}
		for i := 0; i < v.Len(); i++ {
			vv := v.Index(i)
			err := s.Add(vv.Interface())
			if err != nil {
				return err
			}
		}
		return nil
	}

	s2 := setb.ToThreadUnsafe().(*set)
	s2.Range(func(_ int, elem interface{}) bool {
		s.Add(elem)
		return true
	})

	return nil
}

func (s *set) Copy() Set {
	c := &set{
		typedSetGroup: s.typedSetGroup.Copy(),
	}
	return c
}

func (s *set) Equal(b Set) bool {
	s2 := b.ToThreadUnsafe().(*set)
	return s.typedSetGroup.Equal(s2.typedSetGroup)
}

func (s *set) IsSubsetOf(b Set) bool {
	s2 := b.ToThreadUnsafe().(*set)
	return s.typedSetGroup.IsSubsetOf(s2.typedSetGroup)
}

func (s *set) IsSupersetOf(b Set) bool {
	s2 := b.ToThreadUnsafe().(*set)
	return s2.typedSetGroup.IsSubsetOf(s.typedSetGroup)
}

func (s *set) String() string {
	buf := bytes.Buffer{}
	buf.WriteString("Set[")
	s.Range(func(i int, elem interface{}) bool {
		if i == 0 {
			buf.WriteString(fmt.Sprintf("%+v", elem))
		} else {
			buf.WriteString(fmt.Sprintf(" %+v", elem))
		}
		return true
	})
	buf.WriteString("]")
	return buf.String()
}

func (s *set) ToThreadUnsafe() Set {
	return s
}

func (s *set) ToThreadSafe() Set {
	return &threadSafeSet{unsafe: s}
}

func (s *set) Diff(b Set) Set {
	s2 := b.ToThreadUnsafe().(*set)
	diff := &set{
		typedSetGroup: s.typedSetGroup.Diff(s2.typedSetGroup),
	}
	return diff
}

func (s *set) SymmetricDiff(b Set) Set {
	s2 := b.ToThreadUnsafe().(*set)
	diff := &set{
		typedSetGroup: s.typedSetGroup.SymmetricDiff(s2.typedSetGroup),
	}
	return diff
}

func (s *set) Unite(b Set) Set {
	s2 := b.ToThreadUnsafe().(*set)
	union := &set{
		typedSetGroup: s.typedSetGroup.Unite(s2.typedSetGroup),
	}
	return union
}

func (s *set) Intersect(b Set) Set {
	s2 := b.ToThreadUnsafe().(*set)
	intersection := &set{
		typedSetGroup: s.typedSetGroup.Intersect(s2.typedSetGroup),
	}
	return intersection
}
