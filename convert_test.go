package lipglossc

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/kr/pretty"
	"github.com/pmezard/go-difflib/difflib"
)

func TestImport(t *testing.T) {
	emptyStyle := lipgloss.NewStyle()
	td := []struct {
		src    lipgloss.Style
		in     string
		out    string
		expErr string
	}{
		{emptyStyle, ``, ``, ``},
		{emptyStyle.PaddingLeft(11), `padding-left:22`, `padding-left: 22;`, ``},
		{emptyStyle, `bold: true`, `bold: true;`, ``},
		{emptyStyle, `bold: true extra`, ``, `in "bold: true extra": excess values at end: ...extra`},
		{emptyStyle.Foreground(lipgloss.Color("11")), `foreground: unset`, ``, ``},
		{emptyStyle, `align: top`, ``, ``},
		{emptyStyle, `align: bottom`, `align: 1;`, ``},
		{emptyStyle, `align: center`, `align: 0.5;`, ``},
		{emptyStyle, `align: left`, ``, ``},
		{emptyStyle, `align: right`, `align: 1;`, ``},
		{emptyStyle.Foreground(lipgloss.Color("11")), `foreground: none`, ``, ``},
		{emptyStyle, `foreground: 11`, `foreground: 11;`, ``},
		{emptyStyle, `foreground: #123`, `foreground: #123;`, ``},
		{emptyStyle, `foreground: #123456`, `foreground: #123456;`, ``},
		{emptyStyle, `foreground: #axxa`, ``, `in "foreground: #axxa": color not recognized`},
		{emptyStyle, `foreground: adaptive(1,2)`, `foreground: adaptive(1,2);`, ``},
		{emptyStyle, `foreground: adaptive(a,b)`, ``, `in "foreground: adaptive(a,b)": color not recognized: "a"`},
		{emptyStyle, `border-style: border("","","","","","","","")`, ``, ``},
		{emptyStyle,
			`border-style: border("a","b","c","d","e","f","g","h")`,
			`border-style: border("a","b","c","d","e","f","g","h");`, ``},
		{emptyStyle,
			`border-style: border("\"","\x41","\102","\u0041","\U00000041","abc","a\"b","\\")`,
			`border-style: border("\"","A","B","A","A","abc","a\"b","\\");`, ``},
		{emptyStyle,
			`border: border("a","b","c","d","e","f","g","h") true false`,
			`border-bottom: true;
border-bottom-size: 1;
border-style: border("a","b","c","d","e","f","g","h");
border-top: true;
border-top-width: 1;`, ``},
	}

	for i, tc := range td {
		t.Run(fmt.Sprintf("%d: %s", i, tc.in), func(t *testing.T) {
			result, err := Import(tc.src, tc.in)
			if err != nil {
				if tc.expErr != "" {
					if err.Error() != tc.expErr {
						t.Fatalf("expected error:\n%q\ngot:\n%q", tc.expErr, err)
					}
					return
				} else {
					t.Fatal(err)
				}
			}
			if tc.expErr != "" {
				t.Fatalf("expected error %q, got no error", tc.expErr)
			}
			t.Logf("%# v", pretty.Formatter(result))
			actual := Export(result, WithSeparator("\n"))
			if actual != tc.out {
				expectedLines := difflib.SplitLines(tc.out)
				actualLines := difflib.SplitLines(actual)
				diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
					Context: 5,
					A:       expectedLines,
					B:       actualLines,
				})
				if err != nil {
					t.Fatal(err)
				}

				t.Fatalf("mismatch:\n%s\ndiff:\n%s", actual, diff)
			}
		})
	}
}

func TestExport(t *testing.T) {
	style := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		BorderTopForeground(lipgloss.Color("12")).
		PaddingTop(2).
		PaddingLeft(4).
		Width(22)

	t.Run("shortened", func(t *testing.T) {
		exp := `align: 0.5;
background: #7D56F4;
bold: true;
border-top-foreground: 12;
foreground: #FAFAFA;
padding-left: 4;
padding-top: 2;
width: 22;`
		result := Export(style, WithSeparator("\n"))
		if result != exp {
			expectedLines := difflib.SplitLines(exp)
			actualLines := difflib.SplitLines(result)
			diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				Context: 5,
				A:       expectedLines,
				B:       actualLines,
			})
			if err != nil {
				t.Fatal(err)
			}

			t.Errorf("mismatch:\n%s\ndiff:\n%s", result, diff)
		}
	})

	t.Run("full", func(t *testing.T) {
		exp := `align: 0.5;
background: #7D56F4;
blink: false;
bold: true;
border-bottom: false;
border-bottom-background: none;
border-bottom-foreground: none;
border-bottom-size: 0;
border-left: false;
border-left-background: none;
border-left-foreground: none;
border-left-size: 0;
border-right: false;
border-right-background: none;
border-right-foreground: none;
border-right-size: 0;
border-style: border("","","","","","","","");
border-top: false;
border-top-background: none;
border-top-foreground: 12;
border-top-width: 0;
color-whitespace: false;
faint: false;
foreground: #FAFAFA;
height: 0;
inline: false;
italic: false;
margin-bottom: 0;
margin-left: 0;
margin-right: 0;
margin-top: 0;
max-height: 0;
max-width: 0;
padding-bottom: 0;
padding-left: 4;
padding-right: 0;
padding-top: 2;
reverse: false;
strikethrough: false;
strikethrough-spaces: false;
underline: false;
underline-spaces: false;
width: 22;`
		result := Export(style, WithExportDefaults(), WithSeparator("\n"))
		if result != exp {
			expectedLines := difflib.SplitLines(exp)
			actualLines := difflib.SplitLines(result)
			diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				Context: 5,
				A:       expectedLines,
				B:       actualLines,
			})
			if err != nil {
				t.Fatal(err)
			}

			t.Errorf("mismatch:\n%s\ndiff:\n%s", result, diff)
		}
	})
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"hello", "Hello"},
		{"hello-world", "HelloWorld"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			res := camelCase(tc.in)
			if res != tc.out {
				t.Errorf("expected %q, got %q", tc.out, res)
			}
		})
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"Hello", "hello"},
		{"HelloWorld", "hello-world"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			res := snakeCase(tc.in)
			if res != tc.out {
				t.Errorf("expected %q, got %q", tc.out, res)
			}
		})
	}
}
