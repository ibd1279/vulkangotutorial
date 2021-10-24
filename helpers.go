package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"

	vk "github.com/vulkan-go/vulkan"
)

// Error handling
func MustSucceed(result vk.Result) {
	err := vk.Error(result)
	if err != nil {
		panic(err)
	}
}

// Optional Uint32
type OptionUint32 struct {
	v   uint32
	set bool
}

func (option *OptionUint32) Set(v uint32) {
	option.v = v
	option.set = true
}

func (option OptionUint32) IsSet() bool {
	return option.set
}

func (option OptionUint32) Val() uint32 {
	if !option.IsSet() {
		panic("Attempt to use option value that hasn't been set.")
	}
	return option.v
}

// To C Strings
func ToCString(input string) string {
	l := len(input)
	if l == 0 {
		return "\x00"
	} else if input[l-1] != '\x00' {
		return fmt.Sprintf("%s\x00", input)
	}
	return input
}

func ToCStrings(input []string) []string {
	a := make([]string, len(input))
	for k, v := range input {
		a[k] = ToCString(v)
	}
	return a
}

// Checking Support
func SliceToMap(keys []string) map[string]bool {
	output := make(map[string]bool)
	for _, v := range keys {
		output[ToCString(v)] = true
	}
	return output
}

func SetSubtraction(a []string, b map[string]bool) []string {
	output := make([]string, 0)
	for _, v := range a {
		if !b[ToCString(v)] {
			output = append(output, v)
		}
	}
	return output
}

func DedupeSlice(a []string) []string {
	idx := make(map[string]bool)
	output := make([]string, 0)
	for _, v := range a {
		if !idx[ToCString(v)] {
			output = append(output, v)
			idx[ToCString(v)] = true
		}
	}
	return output
}

func MustSupport(available, required []string) {
	missing := SetSubtraction(required, SliceToMap(available))
	if len(missing) > 0 {
		err := fmt.Errorf("Required values %v not found in %v.", missing, available)
		panic(err)
	}
}

// Clamp-able
func ClampUint32(v, smallest, largest uint32) uint32 {
	return MaxUint32(smallest, MinUint32(v, largest))
}

func MaxUint32(x, y uint32) uint32 {
	if x < y {
		return y
	}
	return x
}

func MinUint32(x, y uint32) uint32 {
	if x > y {
		return y
	}
	return x
}

// Must read a file
func MustReadFile(fn string) []byte {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		panic(err)
	}
	return b
}

// 32-bit Words
type WordsUint32 []uint32

func NewWordsUint32(b []byte) WordsUint32 {
	r := bytes.NewReader(b)
	words := make([]uint32, len(b)/4)
	binary.Read(r, binary.LittleEndian, words)
	return WordsUint32(words)
}

func (words WordsUint32) Sizeof() uint {
	return uint(len(words) * 4)
}
