# lipgloss-convert

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/lipgloss-convert)
[![Build Status](https://github.com/knz/lipgloss-convert/workflows/build/badge.svg)](https://github.com/knz/lipgloss-convert/actions)
[![Go ReportCard](https://goreportcard.com/badge/knz/lipgloss-convert)](https://goreportcard.com/report/knz/lipgloss-convert)
[![Coverage Status](https://coveralls.io/repos/github/knz/lipgloss-convert/badge.svg)](https://coveralls.io/github/knz/lipgloss-convert)

String conversion functions for [lipgloss](https://github.com/charmbracelet/lipgloss) Styles.

This library defines the following two functions:

```go
type Style = lipgloss.Style

// Import reads style specifications from the input string
// and sets the corresponding properties in the dst style.
func Import(dst Style, input string) (Style, error)

// Export emits style specifications that represent
// the given style.
func Export(s Style) string
```

## Exporting styles to text

For example:

```go
import (
   "fmt"

   "github.com/charmbracelet/lipgloss"
   lipglossc "github.com/knz/lipgloss-convert"
)

func main() {
    style := lipgloss.NewStyle().
        Bold(true).
        Align(lipgloss.Center).
        Foreground(lipgloss.Color("#FAFAFA")).
        Background(lipgloss.AdaptiveColor{"#7D56F4", "#112233"})).
        BorderTopForeground(lipgloss.Color("12")).
        BorderStyle(lipgloss.RoundedBorder()).
        PaddingTop(2).
        PaddingLeft(4).
        Width(22)

    fmt.Println(lipglossc.Export(s))
}
```

Displays:

``` css
align: 0.5;
background: adaptive(#7D56F4,#112233);
bold: true;
border-style: border("─","─","│","│","╭","╮","╯","╰");
border-top-foreground: 12;
foreground: #FAFAFA;
padding-left: 4;
padding-top: 2;
width: 22;
```

## Importing styles from text

The `Import` function applies the text directives specified in its input
argument to the style also provided as argument. Other properties already
in the style remain unchanged.

Which properties are supported? See the [lipgloss
documentation](https://pkg.go.dev/github.com/charmbracelet/lipgloss)
for details. `Import` automatically supports all the lipgloss
properties, as follows:

- `Foreground` in lipgloss becomes `foreground` in the textual syntax.
- `UnderlineSpaces` becomes `underline-spaces`.
- etc.

`Import` also supports the following special cases:

- For colors:

  ```
  foreground: #abc;
  foreground: #aabbcc;
  foreground: 123;
  foreground: adaptive(<color>,<color>);
  ```

- Padding, margin etc which can take multiple values at once:

  ```
  margin: 10
  margin: 10 20
  margin: 10 20 10 20
  ```

- Border styles:

  ```
  border-style: rounded;
  border-style: hidden;
  border-style: normal;
  border-style: thick;
  border-style: double;
  ```

- Border styles with top/bottom or left/right selection (see the doc
  for `lipgloss.Style`'s `Border()` method):

  ```
  border-style: normal true false;
  border-style: normal true false false true;
  ```

- Resetting a style with `clear`: this erases all the properties
  in the style, to start with a fresh style.
