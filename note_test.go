package changelog

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_NotesToString(t *testing.T) {
	notes := []Note{
		{
			Type:  "warning",
			Body:  "warning body",
			Issue: "1",
		},
		{
			Type:  "feature",
			Body:  "feature body",
			Issue: "1",
		},
	}

	// backticks can't be escaped in a multiline string, so use tilde in place of backtick and replace during test
	expected := strings.ReplaceAll(`~~~release-note:warning
warning body
~~~

~~~release-note:feature
feature body
~~~
`, "~~~", "```")

	actual := NotesToString(notes)

	if !cmp.Equal(expected, actual) {
		t.Fatal(cmp.Diff(expected, actual))
	}
}
