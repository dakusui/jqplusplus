package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Precompiled regular expression
var parseNodeUnitRegex = regexp.MustCompile(`^([^;]+)(?:;([^;]+)(?:;(.+))?)?$`)

type NodeUnit struct {
	Name    string
	Decoder string
	Args    []string
}

func (n NodeUnit) Hash() string {
	return n.String()
}

func (n NodeUnit) String() string {
	return fmt.Sprintf("%s|%s|%s", n.Name, n.Decoder, fmt.Sprint(n.Args))
}

// Equal method to compare two NodeUnit values
func (n NodeUnit) Equal(other NodeUnit) bool {
	// First compare Name and Decoder fields (string fields are directly comparable)
	if n.Name != other.Name || n.Decoder != other.Decoder {
		return false
	}

	// Compare Args slices; check if sizes are different first
	if len(n.Args) != len(other.Args) {
		return false
	}

	// Create sorted copies of Args slices
	sortedArgs1 := append([]string{}, n.Args...)
	sortedArgs2 := append([]string{}, other.Args...)
	sort.Strings(sortedArgs1)
	sort.Strings(sortedArgs2)

	// Compare the sorted slices using DeepEqual
	return reflect.DeepEqual(sortedArgs1, sortedArgs2)
}

func ParseNodeUnit(e string) (*NodeUnit, error) {
	matches := parseNodeUnitRegex.FindStringSubmatch(e)
	if matches == nil {
		return nil, fmt.Errorf("invalid node unit string: %s", e)
	}
	ret := NodeUnit{
		Name:    matches[1],
		Decoder: "",
		Args:    []string{},
	}
	if len(matches) > 2 {
		ret.Decoder = matches[2]
	}
	if len(matches) > 3 {
		ret.Args = strings.Split(matches[3], ":")
	}
	return &ret, nil
}
