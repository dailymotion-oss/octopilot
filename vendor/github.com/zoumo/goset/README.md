# goset
[![Go Report Card](https://goreportcard.com/badge/github.com/zoumo/goset)](https://goreportcard.com/report/github.com/zoumo/goset)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/zoumo/goset)
[![Coverage Status](https://coveralls.io/repos/github/zoumo/goset/badge.svg?branch=master)](https://coveralls.io/github/zoumo/goset?branch=master)
[![Build Status](https://travis-ci.org/zoumo/goset.svg?branch=master)](https://travis-ci.org/zoumo/goset)


Set is a useful collection but there is no built-in implementation in Go lang.

## Why?

The only one pkg which provides set operations now is [golang-set](https://github.com/deckarep/golang-set)

Unfortunately, the api of golang-set is not good enough.

For example, I want to generate a set from a int slice

```go
import "github.com/deckarep/golang-set"

func main() {
	ints := []int{1, 2, 3, 4}
	mapset.NewSet(ints...)
	mapset.NewSetFromSlice(ints)
	mapset.NewSetWith(ints...)
}
```

the code above can not work, according to

>    cannot use ints (type []int) as type []interface{}

You can not assign any slice to an `[]interface{}`  in Go lang.

>   https://github.com/golang/go/wiki/InterfaceSlice

So you need to copy your elements from `[]int` to `[]interface` by a loop.

That means you must do this manually every time you want to generate a set from slice.

**It is ugly. So I create my own set**

## Usage

```go
import "github.com/zoumo/goset"

func main() {
	goset.NewSet(1, 2, 3, 4)
	// or
	goset.NewSetFrom([]int{1, 2, 3, 4})

	goset.NewSet("1", "2", "3")
	// or
	goset.NewSetFrom([]string{"1", "2", "3"})
}
```

Full API

```go
// Set provides a collection of operations for sets
//
// The implementation of Set is base on hash table. So the elements must be
// hashable, functions, maps, slices are unhashable type, adding these elements
// will cause panic.
//
// There are two implementations of Set:
// 1. default is unsafe based on hash table(map)
// 2. thread safe based on sync.RWMutex
//
// The two kinds of sets can easily convert to the other one. But you must know
// exactly what you are doing to avoid the concurrent race
type Set interface {
	SetToSlice
	// Add adds all given elements to the set anyway, no matter if it whether already exists.
	//
	Add(elem ...interface{}) error

	// Extend adds all elements in the given interface b to this set
	// the given interface must be array, slice or Set.
	Extend(b interface{}) error

	// Remove deletes all given elements from the set.
	Remove(elem ...interface{})

	// Contains checks whether the given elem is in the set.
	Contains(elem interface{}) bool

	// ContainsAll checks whether all the given elems are in the set.
	ContainsAll(elems ...interface{}) bool

	ContainsAny(elems ...interface{}) bool

	// Copy clones the set.
	Copy() Set

	// Len returns the size of set. aka Cardinality.
	Len() int

	// String returns the string representation of the set.
	String() string

	// Range calls f sequentially for each element present in the set.
	// If f returns false, range stops the iteration.
	//
	// Note: the iteration order is not specified and is not guaranteed
	// to be the same from one iteration to the next. The index only
	// means how many elements have been visited in the iteration, it not
	// specifies the index of an element in the set
	Range(foreach func(index int, elem interface{}) bool)

	// ---------------------------------------------------------------------
	// Convert

	// ToThreadUnsafe returns a thread unsafe set.
	// Carefully use the method.
	ToThreadUnsafe() Set

	// ToThreadSafe returns a thread safe set.
	// Carefully use the method.
	ToThreadSafe() Set

	// ---------------------------------------------------------------------
	// Compare

	// Equal checks whether this set is equal to the given one.
	// There are two constraints if set a is equal to set b.
	// the two set must have the same size and contain the same elements.
	Equal(b Set) bool

	// IsSubsetOf checks whether this set is the subset of the given set
	// In other words, all elements in this set are also the elements
	// of the given set.
	IsSubsetOf(b Set) bool

	// IsSupersetOf checks whether this set is the superset of the given set
	// In other words, all elements in the given set are also the elements
	// of this set.
	IsSupersetOf(b Set) bool

	// ---------------------------------------------------------------------
	// Set Oprations

	// Diff returns the difference between the set and this given
	// one, aka Difference Set
	// math formula: a - b
	Diff(b Set) Set

	// SymmetricDiff returns the symmetric difference between this set
	// and the given one. aka Symmetric Difference Set
	// math formula: (a - b) ∪ (b - a)
	SymmetricDiff(b Set) Set

	// Unite combines two sets into a new one, aka Union Set
	// math formula: a ∪ b
	Unite(b Set) Set

	// Intersect returns the intersection of two set, aka Intersection Set
	// math formula: a ∩ b
	Intersect(b Set) Set
}

// SetToSlice contains methods that knows how to convert set to slice.
type SetToSlice interface {
	// ToStrings returns all string elements in this set.
	ToStrings() []string
	// ToInts returns all int elements in this set.
	ToInts() []int
	// Elements returns all elements in this set.
	Elements() []interface{}
}
```

