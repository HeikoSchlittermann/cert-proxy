package list

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

// AddItemsFromFile reads a file, and adds each line as an
// item to the Items list. Comments (#) are allowed.
func AddItemsFromFile(items List, filename string) error {
	if filename == "" {
		return nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return AddItemsFromReader(items, file)
}
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

// Type List implents a ordered list
type OrderedStrings []string

func (list *OrderedStrings) Add(v ...string) {
	*list = append(*list, v...)
}
func (list OrderedStrings) Items() []string {
	sort.Strings(list)
	return list
}

// Type UniqList implements a list of uniq items
type UniqStrings map[string]interface{}

func (list *UniqStrings) Add(v ...string) {
	for _, v := range v {
		(*list)[v] = nil
	}
}
func (list UniqStrings) Items() []string {
	var items = make([]string, 0, len(list))
	for k, _ := range list {
		items = append(items, k)
	}
	return items
}

// String returns the string representation of the list
func (list UniqStrings) String() string {
	return fmt.Sprint(list.Items())
}
