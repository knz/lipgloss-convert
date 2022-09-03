package lipglossc

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// S is a handy alias to simplify declarations in this library.
type S = lipgloss.Style

// Import reads style specifications from the input string
// and sets the corresponding properties in the dst style.
func Import(dst S, input string) (S, error) {
	// Syntax: semicolon-separated list of prop: values... pairs.
	assignments := strings.Split(input, ";")
	for _, a := range assignments {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}

		if a == "clear" {
			// Special keyword: reset style.
			dst = lipgloss.NewStyle()
			continue
		}

		pair := strings.SplitN(a, ":", 2)
		if len(pair) != 2 {
			return dst, fmt.Errorf("invalid syntax: %q", a)
		}
		propName, args := pair[0], pair[1]
		propName = strings.TrimSpace(propName)
		args = strings.TrimSpace(args)
		p, err := getProp(propName)
		if err != nil {
			return dst, fmt.Errorf("in %q: %v", a, err)
		}

		dst, err = p.assign(dst, args)
		if err != nil {
			return dst, fmt.Errorf("in %q: %v", a, err)
		}
	}
	return dst, nil
}

type options struct {
	includeDefaults bool
	sep             string
}

type ExportOption func(*options)

// WithSeparator sets the separator between directives.
func WithSeparator(sep string) ExportOption {
	return func(e *options) {
		e.sep = sep
	}
}

// WithExportDefaults includes the fields that are set to default values.
func WithExportDefaults() ExportOption {
	return func(e *options) {
		e.includeDefaults = true
	}
}

// Export emits style specifications that represent
// the given style.
// If includeDefaults is set, all the fields set to
// default values are also included in the output.
func Export(s S, opts ...ExportOption) string {
	opt := options{
		sep: " ",
	}
	for _, o := range opts {
		o(&opt)
	}

	var buf strings.Builder

	v := reflect.ValueOf(s)
	t := v.Type()
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "Get") {
			continue
		}
		if ignoredMethods[m.Name] {
			continue
		}
		if m.Type.NumIn() != 1 {
			// Method with parameters; not truly a Getter. Ignore.
			continue
		}

		res := m.Func.Call([]reflect.Value{v})

		if !opt.includeDefaults && len(res) == 1 && isDefault(res[0]) {
			// Default value. Don't report anything for this getter.
			continue
		}

		if buf.Len() > 0 {
			buf.WriteString(opt.sep)
		}
		buf.WriteString(snakeCase(strings.TrimPrefix(m.Name, "Get")))
		buf.WriteString(": ")
		for j, v := range res {
			if j > 0 {
				buf.WriteByte(' ')
			}
			printValue(&buf, v)
		}
		buf.WriteByte(';')
	}
	return buf.String()
}

func printValue(buf *strings.Builder, v reflect.Value) {
	switch v.Type().Name() {
	case "TerminalColor":
		tc := v.Interface().(lipgloss.TerminalColor)
		switch c := tc.(type) {
		case lipgloss.NoColor:
			buf.WriteString("none")
		case lipgloss.Color:
			buf.WriteString(string(c))
		case lipgloss.AdaptiveColor:
			fmt.Fprintf(buf, "adaptive(%s,%s)", c.Light, c.Dark)
		default:
			r, g, b, _ := tc.RGBA()
			fmt.Fprintf(buf, "#%02x%02x%02x", r, g, b)
		}
	case "Border":
		b := v.Interface().(lipgloss.Border)
		fmt.Fprintf(buf, "border(%q,%q,%q,%q,%q,%q,%q,%q)",
			b.Top, b.Bottom, b.Left, b.Right,
			b.TopLeft, b.TopRight, b.BottomRight, b.BottomLeft,
		)
	default:
		fmt.Fprintf(buf, "%v", v.Interface())
	}
}

func isDefault(v reflect.Value) bool {
	if v.IsZero() {
		return true
	}
	switch v.Type().Name() {
	case "TerminalColor":
		color := v.Interface().(lipgloss.TerminalColor)
		_, isNoColor := color.(lipgloss.NoColor)
		return isNoColor
	default:
		// Unknown type. Always include in output.
		return false
	}
}

var ignoredMethods = map[string]bool{
	"GetBorder":               true,
	"GetMargin":               true,
	"GetPadding":              true,
	"GetFrameSize":            true,
	"GetVerticalFrameSize":    true,
	"GetHorizontalFrameSize":  true,
	"GetVerticalMargins":      true,
	"GetHorizontalMargins":    true,
	"GetVerticalPadding":      true,
	"GetHorizontalPadding":    true,
	"GetVerticalBorderSize":   true,
	"GetHorizontalBorderSize": true,
}

func getProp(name string) (prop, error) {
	p, ok := propRegistry[name]
	if !ok {
		var err error
		p, err = discoverProp(name)
		if err != nil {
			return prop{}, err
		}
	}
	return p, nil
}

func discoverProp(name string) (prop, error) {
	if strings.HasPrefix(name, "set-") {
		return prop{}, fmt.Errorf("don't use 'set-xx: foo;'  use 'xx: foo;' instead")
	}
	if strings.HasPrefix(name, "unset-") {
		return prop{}, fmt.Errorf("don't use 'unset-xx: foo;' use 'xx: unset;' instead")
	}
	if strings.HasPrefix(name, "get-") {
		return prop{}, fmt.Errorf("property not supported: %q", name)
	}
	propName := name
	name = camelCase(name)
	s := reflect.ValueOf(lipgloss.NewStyle())
	t := s.Type()

	m, hasMethod := t.MethodByName(name)
	if !hasMethod {
		return prop{}, fmt.Errorf("property not supported: %q", propName)
	}
	if m.Type.NumOut() != 1 || m.Type.Out(0) != styleType {
		return prop{}, fmt.Errorf("method %q exists but does not return Style", name)
	}

	var args []argtype
	for i := 1; i < m.Type.NumIn(); i++ {
		argT := m.Type.In(i)

		if m.Type.IsVariadic() && i == m.Type.NumIn()-1 {
			argT = argT.Elem()
		}

		switch {
		case argT.Kind() == reflect.Int:
			args = append(args, inttype{})
		case argT.Kind() == reflect.Bool:
			args = append(args, booltype{})
		case argT.Name() == "Border":
			args = append(args, bordertype{})
		case argT.Name() == "Position":
			args = append(args, postype{})
		case argT.Name() == "TerminalColor":
			args = append(args, colortype{})
		default:
			return prop{}, fmt.Errorf("lipgloss.Style has method %s, but method uses unsupported argument type %s", name, argT)
		}
	}
	p := prop{
		setFn:      m.Func,
		isVariadic: m.Type.IsVariadic(),
		args:       args,
	}

	if um, hasUnsetMethod := t.MethodByName("Unset" + name); hasUnsetMethod &&
		m.Type.NumOut() == 1 && m.Type.Out(0) == styleType {
		p.unsetFn = um.Func
	}

	return p, nil
}

var styleType = reflect.TypeOf(lipgloss.NewStyle())

type argtype interface {
	parse([]byte, int) (int, reflect.Value, error)
}

type inttype struct{}

func (inttype) parse(input []byte, first int) (pos int, val reflect.Value, err error) {
	pos = first
	r := reInt.FindSubmatch(input[pos:])
	if r == nil {
		return pos, val, fmt.Errorf("no value found")
	}
	pos += len(r[0])
	i, err := strconv.Atoi(string(r[1]))
	if err != nil {
		return pos, val, err
	}
	return pos, reflect.ValueOf(i), nil
}

var reInt = regexp.MustCompile(`^\s*([0-9]+)(?:\s+|$)`)

type booltype struct{}

func (booltype) parse(input []byte, first int) (pos int, val reflect.Value, err error) {
	pos = first
	r := reBool.FindSubmatch(input[pos:])
	if r == nil {
		return pos, val, fmt.Errorf("no value found")
	}
	pos += len(r[0])
	b, err := strconv.ParseBool(string(r[1]))
	if err != nil {
		return pos, val, err
	}
	return pos, reflect.ValueOf(b), nil
}

var reBool = regexp.MustCompile(`^\s*(1|[tT]|TRUE|[tT]rue|0|[fF]|FALSE|[fF]alse)(?:\s+|$)`)

type postype struct{}

func (postype) parse(input []byte, first int) (pos int, val reflect.Value, err error) {
	pos = first
	r := rePos.FindSubmatch(input[pos:])
	if r == nil {
		return pos, val, fmt.Errorf("no value found")
	}
	pos += len(r[0])
	word := string(r[1])
	switch word {
	case "top":
		val = reflect.ValueOf(lipgloss.Top)
	case "bottom":
		val = reflect.ValueOf(lipgloss.Bottom)
	case "center":
		val = reflect.ValueOf(lipgloss.Center)
	case "left":
		val = reflect.ValueOf(lipgloss.Left)
	case "right":
		val = reflect.ValueOf(lipgloss.Right)
	default:
		p, err := strconv.ParseFloat(word, 64)
		if err != nil {
			return pos, val, err
		}
		position := lipgloss.Position(p)
		val = reflect.ValueOf(position)
	}
	return pos, val, nil
}

var rePos = regexp.MustCompile(`^\s*(top|bottom|center|left|right|1|1\.0|0\.5|\.5|0|0\.0|\.0)(?:\s+|$)`)

type colortype struct{}

func (colortype) parse(input []byte, first int) (pos int, val reflect.Value, err error) {
	pos = first
	// possible syntaxes:
	// - adaptive(X, Y)
	// - one word, either "none", just a number or a RGB value
	if r := reAdaptive.FindSubmatch(input[pos:]); r != nil {
		pos += len(r[0])
		firstValue := strings.TrimSpace(string(r[1]))
		if !reColor.MatchString(firstValue) {
			return pos, val, fmt.Errorf("color not recognized: %q", firstValue)
		}
		secondValue := strings.TrimSpace(string(r[2]))
		if !reColor.MatchString(secondValue) {
			return pos, val, fmt.Errorf("color not recognized: %q", secondValue)
		}
		c := lipgloss.AdaptiveColor{Light: firstValue, Dark: secondValue}
		val = reflect.ValueOf(c)
		return pos, val, nil
	}

	r := reColorOrNone.FindSubmatch(input[pos:])
	if r == nil {
		return pos, val, fmt.Errorf("color not recognized")
	}
	pos += len(r[0])
	word := string(r[1])
	switch word {
	case "none":
		val = reflect.ValueOf(lipgloss.NoColor{})
	default:
		if !reColor.MatchString(word) {
			return pos, val, fmt.Errorf("color not recognized: %q", word)
		}
		val = reflect.ValueOf(lipgloss.Color(word))
	}
	return pos, val, nil
}

var reColor = regexp.MustCompile(`^\s*(\d+|#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6})(?:\s+|$)`)
var reColorOrNone = regexp.MustCompile(`^\s*(none|\d+|#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6})(?:\s+|$)`)
var reAdaptive = regexp.MustCompile(`^\s*(?:adaptive\s*\(([^,]*),([^,]*)\))(?:\s+|$)`)

type bordertype struct{}

func (bordertype) parse(input []byte, first int) (pos int, val reflect.Value, err error) {
	pos = first
	if r := reSpecialBorder.FindSubmatch(input[pos:]); r != nil {
		pos += len(r[0])
		word := string(r[1])
		var b lipgloss.Border
		switch word {
		case "rounded":
			b = lipgloss.RoundedBorder()
		case "normal":
			b = lipgloss.NormalBorder()
		case "thick":
			b = lipgloss.ThickBorder()
		case "hidden":
			b = lipgloss.HiddenBorder()
		case "double":
			b = lipgloss.DoubleBorder()
		default:
			return pos, val, fmt.Errorf("unrecognized border name: %q", word)
		}
		return pos, reflect.ValueOf(b), nil
	}
	r := reBorder.FindSubmatch(input[pos:])
	if r == nil {
		return pos, val, fmt.Errorf("no valid border value found")
	}
	pos += len(r[0])
	var b lipgloss.Border
	for i, field := range []*string{
		&b.Top, &b.Bottom, &b.Left, &b.Right,
		&b.TopLeft, &b.TopRight, &b.BottomRight, &b.BottomLeft,
	} {
		word := string(r[i+1])
		word, err := strconv.Unquote(word)
		if err != nil {
			return pos, val, err
		}
		*field = word
	}
	val = reflect.ValueOf(b)
	return pos, val, nil
}

// Example valid border strings:
// "h", "|", etc
// "\"" - the character '"' itself
// "\\" - the character '\' itself
// "\012" - a octal-encoded ascii value
// "\xFF" - a hex-encoded ascii value
// "\u1234" - a hex-encoded rune
// "\U12345678" - a hex-encoded rune
var reBorderStr = `"(?:\\[\\"]|\\[0-7]{3}|\\x[0-9a-fA-F]{2}|\\u[0-9]{4}|\\U[0-9]{8}|[^\\"])*"`

var reBorder = regexp.MustCompile(`^\s*(?:border\s*\(\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*,\s*(` +
	reBorderStr + `)\s*\))(?:\s+|$)`)

var reSpecialBorder = regexp.MustCompile(`^\s*(rounded|normal|thick|hidden|double)(?:\s+|$)`)

// camelCase converts hello-world to HelloWorld.
func camelCase(s string) string {
	var buf strings.Builder
	cap := true
	for _, r := range s {
		if r == '-' {
			cap = true
			continue
		}
		if cap {
			r = unicode.ToUpper(r)
			cap = false
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// snakeCase converts HelloWorld to hello-world.
func snakeCase(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsUpper(r) {
			if buf.Len() > 0 {
				buf.WriteByte('-')
			}
			r = unicode.ToLower(r)
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

var propRegistry = map[string]prop{}

type prop struct {
	setFn      reflect.Value
	unsetFn    reflect.Value
	isVariadic bool
	args       []argtype
}

func (p prop) assign(dst S, args string) (S, error) {
	if args == "unset" {
		// Special keyword.
		var noValue reflect.Value
		if p.unsetFn == noValue {
			return dst, fmt.Errorf("no unset method defined")
		}
		out := p.unsetFn.Call([]reflect.Value{reflect.ValueOf(dst)})
		return out[0].Interface().(lipgloss.Style), nil
	}

	// Read the arguments from the input string.
	vals := make([]reflect.Value, 0, 1+len(p.args))
	vals = append(vals, reflect.ValueOf(dst))
	pos := 0
	input := []byte(args)
	for i, arg := range p.args {
		if pos >= len(input) {
			if p.isVariadic && i == len(p.args)-1 {
				// It's ok for a variadic arg list to have zero argument.
				break
			}
			return dst, fmt.Errorf("missing value")
		}
		var err error
		var val reflect.Value
		pos, val, err = arg.parse(input, pos)
		if err != nil {
			return dst, err
		}
		vals = append(vals, val)
	}
	if p.isVariadic {
		for pos < len(input) {
			var val reflect.Value
			var err error
			pos, val, err = p.args[len(p.args)-1].parse(input, pos)
			if err != nil {
				return dst, err
			}
			vals = append(vals, val)
		}
	}
	if pos < len(input) {
		return dst, fmt.Errorf("excess values at end: ...%s", string(input[pos:]))
	}

	// Finally call the setter.
	out := p.setFn.Call(vals)
	return out[0].Interface().(lipgloss.Style), nil
}
