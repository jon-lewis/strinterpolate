package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	err := filepath.WalkDir(".",
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".cs") {
				wg.Add(1)
				go processFile(path, &wg)
			}
			return nil
		})

	if err != nil {
		fmt.Println(err)
	}

	wg.Wait()
}

func processFile(path string, wg *sync.WaitGroup) {
	defer wg.Done()

	contents, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	str := string(contents)

	builder := strings.Builder{}

	for {
		i := strings.Index(str, "string.Format")
		if i == -1 {
			break
		}

		if i != 0 {
			builder.WriteString(str[:i])
		}

		j := parenEndIndex(str[i:])
		if j == -1 {
			panic("failed to find ) parenthesis")
		}

		k := i + j + 1
		builder.WriteString(replace(str[i:k]))

		str = str[k:]
	}

	if builder.Len() == 0 {
		return
	}

	builder.WriteString(str)

	err = os.WriteFile(path, []byte(builder.String()), 0644)
	if err != nil {
		fmt.Println("Failed to write file", path, err)
		return
	}
}

func replace(s string) string {
	buffer := "$"

	// string.Format(...)
	str := s[14 : len(s)-1]
	str = strings.TrimSpace(str)
	i := stringEndIndex(str)
	if i == -1 {
		panic("failed to find end bracket")
	}

	f := str[:i+1]
	f = strings.TrimSpace(f)

	// TODO this is a temp fail-safe to prevent corrupting a string in string.Format that contains concatenation
	if containsConcat(str[i+1:]) {
		return s
	}

	pp := params(str[i+1:])

	if len(pp) == 0 {
		return f
	}

	for i, p := range pp {
		if strings.Contains(p, "string.Format") {
			pp[i] = replace(p)
		}
	}

	for pi, p := range pp {
		f = strings.ReplaceAll(f, fmt.Sprintf("{%d", pi), "{"+p)
	}

	buffer += f

	return buffer
}

func containsConcat(str string) bool {
	for _, r := range str {
		if r == '+' {
			return true
		}
		if r == ',' {
			return false
		}
	}
	return false
}

func params(str string) []string {
	// x, "asdf", x.y.z, await func("asdf", pickles)
	p := []string{}
	var t strings.Builder
	for i := 0; i < len(str); i++ {
		// Don't split on commas in these scopes: (), "", [] {}
		r := str[i]
		var fn func(string) int

		switch r {
		case '(':
			fn = parenEndIndex
		case '[':
			fn = squareEndIndex
		case '{':
			fn = curlyEndIndex
		case '"':
			fn = stringEndIndex
		case ',':
			if t.Len() == 0 {
				continue
			}
			p = append(p, t.String())
			t.Reset()
			continue
		case ' ', '\t':
			if t.Len() > 0 {
				t.WriteByte(r)
			}
			continue
		case '\n', '\r':
			continue
		default:
			t.WriteByte(r)
			continue
		}

		// We found a top-level block, let's find the end of this block and copy/paste into our t.
		e := fn(str[i:])
		if e == -1 {
			panic("failed to find end bracket")
		}
		t.WriteString(str[i : i+e+1])
		i += e
	}
	if t.Len() > 0 {
		p = append(p, t.String())
	}
	return p
}

func curlyEndIndex(str string) int {
	return bracketEndIndex(str, '{', '}')
}

func squareEndIndex(str string) int {
	return bracketEndIndex(str, '[', ']')
}

func parenEndIndex(str string) int {
	return bracketEndIndex(str, '(', ')')
}

func bracketEndIndex(str string, s rune, e rune) int {
	stack := []rune{}
	for i, r := range str {
		// In case the syntax in the file is not correct, let's only process so many bytes
		if i > 10000 {
			return -1
		}

		switch r {
		case s:
			stack = append(stack, s)
		case e:
			if len(stack) == 0 {
				return -1
			}
			if len(stack) == 1 {
				return i
			}
			stack = stack[:len(stack)-1]
		}
	}
	return -1
}

func stringEndIndex(str string) int {
	escaped := false
	count := 0
	for i, r := range str {
		// In case the syntax in the file is not correct, let's only process so many bytes
		if i > 10000 {
			return -1
		}

		switch r {
		case '"':
			if escaped {
				escaped = false
				continue
			}
			count += 1
			if count == 2 {
				return i
			}
		case '\\':
			escaped = !escaped
		default:
			escaped = false
		}
	}
	return -1
}
