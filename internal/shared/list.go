package shared

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// Items implemnt a collection we can add to and list
type Items interface {
	Add(string)
	Items() []string
}

// ReadListFromFile reads a file, and adds each line as an
// item to the Items list. Comments (#) are allowed.
func AddItemsFromFile(items Items, filename string) error {
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
func AddItemsFromReader(items Items, r io.Reader) error {
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

// Type OList implents a ordered list
type OList []string

func (list *OList) Add(v string) {
	*list = append(*list, v)
}
func (list OList) Items() []string {
	return (list)
}

// Type UList implements a list of uniq items
type UList map[string]interface{}

func (list *UList) Add(v string) {
	(*list)[v] = nil
}
func (list UList) Items() []string {
	var items = make([]string, 0, len(list))
	for k, _ := range list {
		items = append(items, k)
	}
	return items
}
