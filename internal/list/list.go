// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package list provides a List interface and functions around lists.
package list //nolint:revive // I accept the collision with Go standard(?) list package

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// List implements a collection we can add to and list
type List interface {
	Add(...string)
	Items() []string
}

// AddItemsFromFile wraps AddItemsFromReader
func AddItemsFromFile(items List, filename string) error {
	if filename == "" {
		return nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	if err := AddItemsFromReader(items, file); err != nil {
		return err
	}

	return file.Close()
}

// AddItemsFromReader reads from the reader, and adds each line as an
// item to the Items list. Comments (#) are allowed.
func AddItemsFromReader(items List, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l := strings.Trim(strings.SplitN(scanner.Text(), "#", 2)[0], " \t\r")
		if l == "" {
			continue
		}

		items.Add(l)
	}

	return scanner.Err()
}

// OrderedStrings implements a ordered list
type OrderedStrings []string

// Add adds new items into the list of ordered strings.
// Basically it appends the new item w/o any sorting operations.
func (list *OrderedStrings) Add(v ...string) {
	*list = append(*list, v...)
}

// Items returns the ordered list of strings
func (list OrderedStrings) Items() []string {
	sort.Strings(list)
	return list
}

// UniqStrings implements a list of uniq items
type UniqStrings map[string]interface{}

// Add adds new items to the list of unique strings. If the
// item exists already, it does nothing.
func (list *UniqStrings) Add(v ...string) {
	for _, v := range v {
		(*list)[v] = nil
	}
}

// Items returns the unique strings of the string list.
func (list UniqStrings) Items() []string {
	var items = make([]string, 0, len(list))
	for k := range list {
		items = append(items, k)
	}

	return items
}

// Copy returns a clone of the list.
func (list UniqStrings) Copy() UniqStrings {
	ss := UniqStrings{}
	ss.Add(list.Items()...)

	return ss
}

// String returns the string representation of the list
func (list UniqStrings) String() string {
	return fmt.Sprint(list.Items())
}
