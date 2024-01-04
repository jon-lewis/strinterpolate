package main

import (
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
)

func TestFuncEndIndex(t *testing.T) {
	var tests = []struct {
		input string
		want  int
	}{
		{"", -1},
		{"(", -1},
		{")", -1},
		{"))", -1},
		{"(()", -1},
		{"(())", 3},
		{"(() )", 4},
		{"(()  )", 5},
		{"string.Format(\"{0}\", v)", 22},
	}

	for _, tt := range tests {
		testname := tt.input
		t.Run(testname, func(t *testing.T) {
			ans := parenEndIndex(tt.input)
			if ans != tt.want {
				t.Errorf("got %d, want %d", ans, tt.want)
			}
		})
	}
}

func TestStringEndIndex(t *testing.T) {
	var tests = []struct {
		input string
		want  int
	}{
		{``, -1},
		{`"`, -1},
		{`"\"`, -1},
		{`""`, 1},
		{`"\""`, 3},
		{`"\"asdf\""`, 9},
		{`"asdf", ""`, 5},
	}

	for _, tt := range tests {
		testname := tt.input
		t.Run(testname, func(t *testing.T) {
			ans := stringEndIndex(tt.input)
			if ans != tt.want {
				t.Errorf("got %d, want %d", ans, tt.want)
			}
		})
	}
}

func TestParams(t *testing.T) {
	var tests = []struct {
		input string
		want  []string
	}{
		{
			`x, "asdf", x.y.z, await func("asdf", pickles)`,
			[]string{`x`, `"asdf"`, `x.y.z`, `await func("adsf", pickles)`},
		},
		{
			`, x, "asdf", x.y.z, await func("asdf", pickles)`,
			[]string{`x`, `"asdf"`, `x.y.z`, `await func("adsf", pickles)`},
		},
	}
	for _, tt := range tests {
		testname := tt.input
		t.Run(testname, func(t *testing.T) {
			ans := params(tt.input)

			if slices.Equal(ans, tt.want) {
				t.Errorf("got %v, want %v", ans, tt.want)
			}
		})
	}
}

func TestFiles(t *testing.T) {
	var tests = []struct {
		input string
		want  string
	}{
		{`string.Format("{0}", "a")`, `$"{"a"}"`},
		{`string.Format("{0}", new string[]{"a"})`, `$"{new string[]{"a"}}"`},
		{`string.Format("[{0}] | {1}", myVar, anotherVar)`, `$"[{myVar}] | {anotherVar}"`},
		{`string.Format("{0}: {1}", dt, await http.Response())`, `$"{dt}: {await http.Response()}"`},
		{`string.Format("\"{0}\"", myVar)`, `$"\"{myVar}\""`},
		{`string.Format("{0}, {1}", point.X, point.Y)`, `$"{point.X}, {point.Y}"`},
		{`string.Format("{0}, {1}", x.Substring(0,4), y.Substring(0,6))`, `$"{x.Substring(0,4)}, {y.Substring(0,6)}"`},
		{`string.Format("Got an error\n{0}", err.ExceptionMessage)`, `$"Got an error\n{err.ExceptionMessage}"`},
		{`string.Format("Calling a function with strings: {0}", func("asdf", "cde"))`, `$"Calling a function with strings: {func("asdf", "cde")}"`},
		{`Console.WriteLine(string.Format("{0}, {1}", point.X, point.Y));
var s = "asdf";
var x = "pickles";
var batman = string.Format("{0}\n{1}", s.Substring(0, 420), x.Substring(0,69));`, `Console.WriteLine($"{point.X}, {point.Y}");
var s = "asdf";
var x = "pickles";
var batman = $"{s.Substring(0, 420)}\n{x.Substring(0,69)}";`},
	}

	for i, tt := range tests {
		testname := tt.input
		t.Run(testname, func(t *testing.T) {
			i := i
			tt := tt
			t.Parallel()

			f := fmt.Sprintf("./tests/Test%d.cs", i)
			err := os.WriteFile(f, []byte(tt.input), 0644)
			if err != nil {
				t.Error(err)
			}

			var wg sync.WaitGroup
			wg.Add(1)
			processFile(f, &wg)

			b, err := os.ReadFile(f)
			if err != nil {
				t.Error(err)
			}
			ans := string(b)
			if ans != tt.want {
				t.Errorf("got %s, want %s", ans, tt.want)
			}
		})
	}
}
