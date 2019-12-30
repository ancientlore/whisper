package main

import (
	"bytes"
	"testing"
)

func TestExtractFrontMatter(t *testing.T) {
	var (
		tests = []string{
			``,
			`
		+++
		x = 2
		+++`,
			` ++++++ `,
			`  +++
		 x = "+++"
		 +++
		 hello`,
		}
		expect = [][]string{
			[]string{``, ``},
			[]string{`x = 2`, ``},
			[]string{``, `++++++`},
			[]string{`x = "+++"`, `hello`},
		}
	)
	for i := range tests {
		fm, r := extractFrontMatter([]byte(tests[i]))
		fm = bytes.TrimSpace(fm)
		r = bytes.TrimSpace(r)
		if string(fm) != expect[i][0] || string(r) != expect[i][1] {
			t.Errorf("Expected %#v but got %#v", expect[i], []string{string(fm), string(r)})
		}
	}
}
