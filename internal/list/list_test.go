package list

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqStrings_Add_Single(t *testing.T) {
	us := UniqStrings{}
	us.Add("a")

	assert.Equal(t, []string{"a"}, us.Items())
}

func TestUniqStrings_Add_Multiple(t *testing.T) {
	us := UniqStrings{}
	us.Add("a", "b", "c")

	items := us.Items()
	sort.Strings(items)
	assert.Equal(t, []string{"a", "b", "c"}, items)
}

func TestUniqStrings_Add_Duplicates(t *testing.T) {
	us := UniqStrings{}
	us.Add("a", "b", "a", "c", "b")

	assert.Len(t, us.Items(), 3)
}

func TestUniqStrings_Items_Empty(t *testing.T) {
	us := UniqStrings{}
	assert.Empty(t, us.Items())
}

func TestUniqStrings_Copy(t *testing.T) {
	us := UniqStrings{}
	us.Add("x", "y")
	cp := us.Copy()

	cp.Add("z")
	assert.Len(t, us.Items(), 2, "original must not be affected")
	assert.Len(t, cp.Items(), 3)
}

func TestUniqStrings_String(t *testing.T) {
	us := UniqStrings{}
	us.Add("hello")

	assert.Contains(t, us.String(), "hello")
}

func TestOrderedStrings_Add(t *testing.T) {
	var os OrderedStrings
	os.Add("b")
	os.Add("a")

	assert.Len(t, os, 2)
}

func TestOrderedStrings_Items_Sorted(t *testing.T) {
	var os OrderedStrings
	os.Add("cherry", "apple", "banana")

	assert.Equal(t, []string{"apple", "banana", "cherry"}, os.Items())
}

func TestOrderedStrings_Items_Empty(t *testing.T) {
	var os OrderedStrings
	assert.Len(t, os.Items(), 0)
}

func TestAddItemsFromReader_Normal(t *testing.T) {
	input := "example.com\nsub.example.com\nanother.org\n"
	us := UniqStrings{}

	require.NoError(t, AddItemsFromReader(&us, strings.NewReader(input)))

	items := us.Items()
	sort.Strings(items)
	assert.Equal(t, []string{"another.org", "example.com", "sub.example.com"}, items)
}

func TestAddItemsFromReader_Comments(t *testing.T) {
	input := "# this is a comment\nexample.com\n# another comment\nsub.example.com\n"
	us := UniqStrings{}

	require.NoError(t, AddItemsFromReader(&us, strings.NewReader(input)))
	assert.Len(t, us.Items(), 2)
}

func TestAddItemsFromReader_InlineComments(t *testing.T) {
	input := "example.com # primary domain\nsub.example.com\n"
	us := UniqStrings{}

	require.NoError(t, AddItemsFromReader(&us, strings.NewReader(input)))

	items := us.Items()
	sort.Strings(items)
	assert.Equal(t, "example.com", items[0], "inline comment should be stripped")
}

func TestAddItemsFromReader_BlankLines(t *testing.T) {
	input := "\n\nexample.com\n\n\nsub.example.com\n\n"
	us := UniqStrings{}

	require.NoError(t, AddItemsFromReader(&us, strings.NewReader(input)))
	assert.Len(t, us.Items(), 2)
}

func TestAddItemsFromReader_WhitespaceLines(t *testing.T) {
	input := "  \t  \nexample.com\n   \n"
	us := UniqStrings{}

	require.NoError(t, AddItemsFromReader(&us, strings.NewReader(input)))
	assert.Len(t, us.Items(), 1)
}

func TestAddItemsFromFile_EmptyFilename(t *testing.T) {
	us := UniqStrings{}
	require.NoError(t, AddItemsFromFile(&us, ""))
	assert.Empty(t, us.Items())
}

func TestAddItemsFromFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "domains.txt")
	require.NoError(t, os.WriteFile(path, []byte("example.com\n# comment\nsub.example.com\n"), 0644))

	us := UniqStrings{}
	require.NoError(t, AddItemsFromFile(&us, path))
	assert.Len(t, us.Items(), 2)
}

func TestAddItemsFromFile_MissingFile(t *testing.T) {
	us := UniqStrings{}
	err := AddItemsFromFile(&us, "/nonexistent/path/to/file.txt")
	assert.Error(t, err)
}

func TestAddItemsFromReader_WithOrderedStrings(t *testing.T) {
	input := "banana\napple\ncherry\n"

	var os OrderedStrings

	require.NoError(t, AddItemsFromReader(&os, strings.NewReader(input)))
	assert.Equal(t, []string{"apple", "banana", "cherry"}, os.Items())
}
