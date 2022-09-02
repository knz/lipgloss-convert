# lipgloss-convert

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
        Background(lipgloss.Color("#7D56F4")).
        BorderTopForeground(lipgloss.Color("12")).
        PaddingTop(2).
        PaddingLeft(4).
        Width(22)

    fmt.Println(lipglossc.Export(s))
}
```

Displays:

``` css
align: 0.5;
background: #7D56F4;
bold: true;
border-top-foreground: 12;
foreground: #FAFAFA;
padding-left: 4;
padding-top: 2;
width: 22;
```

Then using the `Import()` function on the result will recover the original `lipgloss.Style`.

See the [lipgloss
documentation](https://pkg.go.dev/github.com/charmbracelet/lipgloss)
for details. This library automatically supports all the lipgloss
properties, as follows:

- `Foreground` in lipgloss becomes `foreground` in the textual syntax.
- `UnderlineSpaces` becomes `underline-spaces`.
- etc.
