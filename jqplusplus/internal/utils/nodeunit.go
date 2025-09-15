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
	name    string
	decoder string
	args    []string
}

func (n NodeUnit) Hash() string {
	return n.String()
}

func (n NodeUnit) String() string {
	return fmt.Sprintf("%s|%s|%s", n.name, n.decoder, fmt.Sprint(n.args))
}

// Equal method to compare two NodeUnit values
func (n NodeUnit) Equal(other NodeUnit) bool {
	// First compare name and decoder fields (string fields are directly comparable)
	if n.name != other.name || n.decoder != other.decoder {
		return false
	}

	// Compare args slices; check if sizes are different first
	if len(n.args) != len(other.args) {
		return false
	}

	// Create sorted copies of args slices
	sortedArgs1 := append([]string{}, n.args...)
	sortedArgs2 := append([]string{}, other.args...)
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
		name:    matches[1],
		decoder: "",
		args:    []string{},
	}
	if len(matches) > 2 {
		ret.decoder = matches[2]
	}
	if len(matches) > 3 {
		ret.args = strings.Split(matches[3], ":")
	}
	return &ret, nil
}
