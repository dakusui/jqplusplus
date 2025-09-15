package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/itchyny/gojq"
	"regexp"
	"strconv"
	"sync"
)

func Find(n any, pexp string) (any, error) {
	path, err := PexpToPath(pexp)
	if err != nil {
		return nil, err
	}
	ret := n
	for _, k := range path {
		switch k.(type) {
		case string:
			switch ret.(type) {
			case map[string]any:
				r, ok := ret.(map[string]any)[k.(string)]
				if !ok {
					return nil, fmt.Errorf("path %q is not an object", pexp)
				}
				ret = r
			default:
				return nil, fmt.Errorf("type mismatch: %T", n)
			}
		case int:
			switch ret.(type) {
			case []any:
				if k.(int) < 0 || k.(int) >= len(ret.([]any)) {
					return nil, fmt.Errorf("index out of range")
				}
				r := ret.([]any)[k.(int)]
				ret = r
			}
			break
		default:
			msg, _ := fmt.Printf("unknown type: %T", k)
			panic(msg)
		}
	}
	return ret, nil

}

func PathToPexp(segments []any) string {
	var b bytes.Buffer
	for _, s := range segments {
		switch v := s.(type) {
		case string:
			// ."key" with proper escaping; jq prints quotes inside
			b.WriteByte('.')
			b.WriteString(strconv.Quote(v))
		case int:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(v))
			b.WriteByte(']')
		case json.Number:
			// array indices are ints in our traversal; this is here just in case
			b.WriteByte('[')
			b.WriteString(v.String())
			b.WriteByte(']')
		default:
			// should not occur; keep behavior defined
			b.WriteByte('.')
			b.WriteString(strconv.Quote(fmtAny(v)))
		}
	}
	return b.String()
}

func fmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// token matches one of:
//   - ."<quoted key>"
//   - .bareIdentifier   (Unicode letters allowed; start with letter/_; continue with letter/number/_)
//   - [digits]
//
// Leading whitespace between tokens is allowed.
var pexpTok = regexp.MustCompile(`^\s*(?:\."((?:\\.|[^"\\])*)"|\.([\p{L}_][\p{L}\p{N}_]*)|\[(\d+)\])`)

// PexpToPath parses a jq-like path expression into a []any path.
// Accepts both quoted and bare keys (jq style) and integer array indices.
// Examples:
//
//	.foo[0]."bar"    => []any{"foo", 0, "bar"}
//	."a\"b"[12]      => []any{"a\"b", 12}
//	."日本語".x       => []any{"日本語", "x"}
//
// Whitespace between tokens is ignored (e.g., .foo  [1]  ."bar")
func PexpToPath(pexp string) ([]any, error) {
	var out []any
	i := 0
	for i < len(pexp) {
		loc := pexpTok.FindStringSubmatchIndex(pexp[i:])
		if loc == nil {
			// If we haven't consumed anything yet, surface a clear error.
			if len(out) == 0 {
				return nil, fmt.Errorf("invalid path expression near byte %d", i)
			}
			// Otherwise, trailing junk/partial token.
			return nil, fmt.Errorf("unexpected input at byte %d", i)
		}
		// Entire match spans [0:loc[1]) of the remaining string.
		keyQuotedStart, keyQuotedEnd := loc[2], loc[3] // group 1
		keyBareStart, keyBareEnd := loc[4], loc[5]     // group 2
		idxStart, idxEnd := loc[6], loc[7]             // group 3

		switch {
		case keyQuotedStart != -1:
			raw := pexp[i+keyQuotedStart : i+keyQuotedEnd]
			// Unescape using JSON rules (same as strconv.Quote produced earlier).
			unq, err := strconv.Unquote(`"` + raw + `"`)
			if err != nil {
				return nil, fmt.Errorf("bad quoted key at byte %d: %w", i, err)
			}
			out = append(out, unq)

		case keyBareStart != -1:
			key := pexp[i+keyBareStart : i+keyBareEnd]
			out = append(out, key)

		case idxStart != -1:
			raw := pexp[i+idxStart : i+idxEnd]
			n, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("bad index at byte %d: %w", i, err)
			}
			out = append(out, n)

		default:
			return nil, fmt.Errorf("internal parse error at byte %d", i)
		}

		i += loc[1] // advance by match length
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("empty path expression")
	}
	return out, nil
}

// The entire jq program (scalars_or_empty + merge logic) in one literal.
// Semantics match your jq-front snippet exactly.
const jqProgram = `
def scalars_or_empty:
  select(. == null or . == true or . == false
         or type == "number" or type == "string"
         or ((type=="array" or type=="object") and length==0));

def value_at($n; $p): $n | getpath($p);

def setvalue_at($n; $p; $v):
  def type_of($v): $v | type;
  def _setvalue_at($n; $p; $v):
    $n | try setpath($p; $v)
         catch error("Failed to process node at path:<\($p)>; the value:<\($v)>).");
  if (type_of($v)=="object" or type_of($v)=="array") then
    if (value_at($n; $p) | type) as $t | ($t != "object" and $t != "array") then
      _setvalue_at($n; $p; $v)
    else
      .
    end
  else
    _setvalue_at($n; $p; $v)
  end;

def merge_objects($a; $b):
  $b | [paths(scalars_or_empty)]
     | reduce .[] as $p ($a; setvalue_at(.; $p; $b | getpath($p)));

merge_objects($a; $b)
`

var (
	compileOnce sync.Once
	code        *gojq.Code
	compileErr  error
)

func ensureCompiled() error {
	compileOnce.Do(func() {
		q, err := gojq.Parse(jqProgram)
		if err != nil {
			compileErr = fmt.Errorf("parse jq: %w", err)
			return
		}
		c, err := gojq.Compile(q, gojq.WithVariables([]string{"a", "b"}))
		if err != nil {
			compileErr = fmt.Errorf("compile jq: %w", err)
			return
		}
		code = c
	})
	return compileErr
}

// MergeObjectNodes merges b into a using jq-front semantics.
// a and b must be JSON objects (map[string]interface{}).
// The result is a NEW map (a is not mutated).
func MergeObjectNodes(a, b map[string]interface{}) (map[string]interface{}, error) {
	if err := ensureCompiled(); err != nil {
		return nil, err
	}
	// Run compiled jq with $a and $b bound; input is null
	iter := code.Run(nil, a, b)

	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("jq returned no result")
	}
	if err, isErr := v.(error); isErr {
		return nil, fmt.Errorf("jq error: %w", err)
	}
	// Normalize to map[string]interface{}
	if m, ok := v.(map[string]interface{}); ok {
		return m, nil
	}
	buf, _ := json.Marshal(v)
	var out map[string]interface{}
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, fmt.Errorf("unexpected jq result type: %T", v)
	}
	return out, nil
}
