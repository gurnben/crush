package styles

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/diffview"
	"github.com/charmbracelet/x/exp/charmtone"
)

// ThemeKeyForProvider returns a stable identifier for the theme
// associated with the given provider ID. Providers that share a theme
// yield the same key, so callers can cheaply detect when switching
// providers would not actually change the active theme and skip the
// expensive style rebuild. This is the single source of truth for the
// provider-to-theme mapping; [ThemeForProvider] builds on it.
func ThemeKeyForProvider(providerID string) string {
	switch providerID {
	case "hyper":
		return "hyper"
	default:
		return "default"
	}
}

// ThemeForProvider returns the Styles associated with the given provider
// ID. Unknown or empty provider IDs yield the default Charmtone Pantera
// theme.
func ThemeForProvider(providerID string) Styles {
	switch ThemeKeyForProvider(providerID) {
	case "hyper":
		return HypercrushObsidiana()
	default:
		return CharmtonePantera()
	}
}

// CharmtonePantera returns the Charmtone dark theme. It's the default style
// for the UI.
func CharmtonePantera() Styles {
	return charmtoneOverrides(quickStyle(charmtoneOpts()))
}

// charmtoneOpts returns the quickStyleOpts for the Charmtone dark theme,
// using colors from the upstream charmbracelet/x/exp/charmtone package.
func charmtoneOpts() quickStyleOpts {
	return quickStyleOpts{
		primary:   charmtone.Charple,
		secondary: charmtone.Dolly,
		accent:    charmtone.Bok,
		keyword:   charmtone.Blush,

		fgBase:       charmtone.Sash,
		fgMoreSubtle: charmtone.Squid,
		fgSubtle:     charmtone.Smoke,
		fgMostSubtle: charmtone.Oyster,

		onPrimary: charmtone.Butter,

		bgBase:         charmtone.Pepper,
		bgLeastVisible: charmtone.BBQ,
		bgLessVisible:  charmtone.Char,
		bgMostVisible:  charmtone.Iron,

		separator: charmtone.Char,

		destructive:       charmtone.Coral,
		error:             charmtone.Sriracha,
		warningSubtle:     charmtone.Zest,
		warning:           charmtone.Mustard,
		attention:         charmtone.Tang,
		busy:              charmtone.Citron,
		info:              charmtone.Malibu,
		infoMoreSubtle:    charmtone.Sardine,
		infoMostSubtle:    charmtone.Damson,
		success:           charmtone.Julep,
		successMoreSubtle: charmtone.Bok,
		successMostSubtle: charmtone.Guac,

		// ANSI 16-color palette for remapping raw terminal output
		// (e.g. bang-mode shell commands) onto legible Charmtone colors.
		ansiBlack:   charmtone.BBQ,
		ansiRed:     charmtone.Coral,
		ansiGreen:   charmtone.Guac,
		ansiYellow:  charmtone.Mustard,
		ansiBlue:    charmtone.Charple,
		ansiMagenta: charmtone.Dolly,
		ansiCyan:    charmtone.Malibu,
		ansiWhite:   charmtone.Smoke,

		ansiBrightBlack:   charmtone.Iron,
		ansiBrightRed:     charmtone.Tuna,
		ansiBrightGreen:   charmtone.Julep,
		ansiBrightYellow:  charmtone.Zest,
		ansiBrightBlue:    charmtone.Guppy,
		ansiBrightMagenta: charmtone.Blush,
		ansiBrightCyan:    charmtone.Sardine,
		ansiBrightWhite:   charmtone.Salt,
	}
}

// charmtoneOverrides applies Charmtone-specific tweaks that don't fit the
// token model of [quickStyleOpts].
func charmtoneOverrides(s Styles) Styles {
	// Bang ! prompt overrides - use Salt/Hazy/Larple colors.
	s.Editor.PromptBangIconFocused = s.Editor.PromptBangIconFocused.
		Foreground(charmtone.Salt).
		Background(charmtone.Hazy)
	s.Editor.PromptBangDotsFocused = s.Editor.PromptBangDotsFocused.
		Foreground(charmtone.Hazy)
	s.Editor.PromptBangDotsBlurred = s.Editor.PromptBangDotsBlurred.
		Foreground(charmtone.Larple)

	// Shell bar/prompt overrides - use Charple/Iron/Hazy colors.
	s.Messages.ShellBarFocused = s.Messages.ShellBarFocused.
		BorderForeground(charmtone.Charple)
	s.Messages.ShellBarBlurred = s.Messages.ShellBarBlurred.
		BorderForeground(charmtone.Iron)
	s.Messages.ShellPrompt = s.Messages.ShellPrompt.
		Foreground(charmtone.Hazy)
	s.Messages.ShellPromptBlurred = s.Messages.ShellPromptBlurred.
		Foreground(charmtone.Hazy)

	// Restore the original Charmtone syntax-highlight and markdown colors
	// where the generic quickStyle token choices diverge from the palette
	// this theme has always used.
	chroma := s.Markdown.CodeBlock.Chroma
	if chroma != nil {
		chroma.CommentPreproc.Color = hex(charmtone.Bengal)
		chroma.KeywordReserved.Color = hex(charmtone.Pony)
		chroma.KeywordNamespace.Color = hex(charmtone.Pony)
		chroma.KeywordType.Color = hex(charmtone.Guppy)
		chroma.Operator.Color = hex(charmtone.Salmon)
		chroma.NameTag.Color = hex(charmtone.Mauve)
		chroma.NameAttribute.Color = hex(charmtone.Hazy)
		chroma.NameClass.Color = hex(charmtone.Salt)
		chroma.LiteralString.Color = hex(charmtone.Cumin)
	}
	s.Markdown.Link.Color = hex(charmtone.Zinc)
	s.Markdown.Image.Color = hex(charmtone.Cheeky)

	return s
}

// HypercrushObsidiana returns the Hypercrush dark theme.
func HypercrushObsidiana() Styles {
	return CharmtonePantera()
}

func orangeCrushOpts() quickStyleOpts {
	return quickStyleOpts{

		primary:   lipgloss.Color("#D96757"),
		secondary: lipgloss.Color("#C9AA4A"),
		accent:    lipgloss.Color("#F7F3ED"),
		keyword:   lipgloss.Color("#B86A4F"),

		// Foreground
		fgBase:       lipgloss.Color("#ECE7DF"),
		fgSubtle:     lipgloss.Color("#B8AEA1"),
		fgMoreSubtle: lipgloss.Color("#90867A"),
		fgMostSubtle: lipgloss.Color("#6D655C"),

		onPrimary: lipgloss.Color("#171412"),

		// Background
		bgBase:         lipgloss.Color("#202020"),
		bgLeastVisible: lipgloss.Color("#202020"),
		bgLessVisible:  lipgloss.Color("#202020"),
		bgMostVisible:  lipgloss.Color("#454545"),

		separator: lipgloss.Color("#45403B"),

		destructive: lipgloss.Color("#D06A62"),
		error:       lipgloss.Color("#D06A62"),

		warningSubtle: lipgloss.Color("#B8844C"),
		warning:       lipgloss.Color("#D7B06A"),
		attention:     lipgloss.Color("#D97757"),

		// Blue complement
		busy:           lipgloss.Color("#6F93D6"),
		info:           lipgloss.Color("#5F86CF"),
		infoMoreSubtle: lipgloss.Color("#4D73B6"),
		infoMostSubtle: lipgloss.Color("#3C5D93"),

		success:           lipgloss.Color("#4BCE96"),
		successMoreSubtle: lipgloss.Color("#4BBE96"),
		successMostSubtle: lipgloss.Color("#4BAE96"),

		// ANSI palette
		ansiBlack:   lipgloss.Color("#24211F"),
		ansiRed:     lipgloss.Color("#D06A62"),
		ansiGreen:   lipgloss.Color("#7AA7D9"),
		ansiYellow:  lipgloss.Color("#D7B06A"),
		ansiBlue:    lipgloss.Color("#5F86CF"),
		ansiMagenta: lipgloss.Color("#B87A63"),
		ansiCyan:    lipgloss.Color("#86B6E8"),
		ansiWhite:   lipgloss.Color("#ECE7DF"),

		ansiBrightBlack:   lipgloss.Color("#6D655C"),
		ansiBrightRed:     lipgloss.Color("#DD8177"),
		ansiBrightGreen:   lipgloss.Color("#94BCE5"),
		ansiBrightYellow:  lipgloss.Color("#E4C17F"),
		ansiBrightBlue:    lipgloss.Color("#81A9F0"),
		ansiBrightMagenta: lipgloss.Color("#D39A7C"),
		ansiBrightCyan:    lipgloss.Color("#A5D1F7"),
		ansiBrightWhite:   lipgloss.Color("#F7F3ED"),
	}
}

// orangeCrushOverrides applies Orange Crush-specific tweaks that don't fit
// the token model of [quickStyleOpts]. It brightens the diff add/remove
// colors to Claude's signature vivid green/red and re-tints the syntax
// highlighting to match Claude's source code palette.
func orangeCrushOverrides(s Styles) Styles {
	// Diff overrides - Claude dark-theme green/red as the line highlight
	// (background), with white line numbers and +/- signs.
	highlightInsert := lipgloss.Color("#0e2705")
	highlightDelete := lipgloss.Color("#370603")
	white := lipgloss.Color("#ffffff")
	onHighlight := lipgloss.Color("#000000")

	s.Diff.InsertLine = diffview.LineStyle{
		LineNumber: lipgloss.NewStyle().
			Foreground(white).
			Background(highlightInsert),
		Symbol: lipgloss.NewStyle().
			Foreground(white).
			Background(highlightInsert),
		Code: lipgloss.NewStyle().
			Foreground(onHighlight).
			Background(highlightInsert),
	}
	s.Diff.DeleteLine = diffview.LineStyle{
		LineNumber: lipgloss.NewStyle().
			Foreground(white).
			Background(highlightDelete),
		Symbol: lipgloss.NewStyle().
			Foreground(white).
			Background(highlightDelete),
		Code: lipgloss.NewStyle().
			Foreground(onHighlight).
			Background(highlightDelete),
	}

	// Syntax-highlight overrides - colors derived from the user's vimrc
	// (xterm 256-color codes converted to hex).
	chroma := s.Markdown.CodeBlock.Chroma
	if chroma != nil {
		// Comments - brighter soft white.
		chroma.Comment.Color = hex(lipgloss.Color("#f2ecdf"))
		// Operators - white.
		chroma.Operator.Color = hex(lipgloss.Color("#ffffff"))
		// Braces/punctuation - white.
		chroma.Punctuation.Color = hex(lipgloss.Color("#ffffff"))
		// Properties - same white as operators.
		chroma.NameAttribute.Color = hex(lipgloss.Color("#ffffff"))
		// Preprocessor (#include/#define) - same as comments.
		chroma.CommentPreproc.Color = hex(lipgloss.Color("#f2ecdf"))
		// Conditionals (if/while/else) - bold, same color as functions.
		chroma.Keyword.Color = hex(lipgloss.Color("#b31853"))
		chroma.Keyword.Bold = new(true)
		chroma.KeywordReserved.Color = hex(lipgloss.Color("#b31853"))
		chroma.KeywordReserved.Bold = new(true)
		// Declaration keywords (def/class) - info blue.
		chroma.KeywordNamespace.Color = hex(lipgloss.Color("#5f86cf"))
		// Types (int/char/void) - green.
		chroma.KeywordType.Color = hex(lipgloss.Color("#b8e898"))
		// None/nil constants - exact RGB(181,134,248).
		chroma.NameConstant.Color = hex(lipgloss.Color("#b586f8"))
		// Variables - white (same as operators/properties).
		chroma.Name.Color = hex(lipgloss.Color("#ffffff"))
		chroma.NameOther.Color = hex(lipgloss.Color("#ffffff"))
		// Strings - exact RGB(228,219,130).
		chroma.LiteralString.Color = hex(lipgloss.Color("#e4db82"))
		// Variables inside strings (interpolation) - white.
		chroma.LiteralStringEscape.Color = hex(lipgloss.Color("#ffffff"))
		// Numbers - deeper purple.
		chroma.LiteralNumber.Color = hex(lipgloss.Color("#9a6ff0"))
		// Statements/booleans - brighter red.
		chroma.Literal.Color = hex(lipgloss.Color("#ff8088"))
		// Functions/methods - green.
		chroma.NameFunction.Color = hex(lipgloss.Color("#b3e053"))
		chroma.NameFunction.Bold = new(true)

		// Response emphasis (bold/italic) - lilac RGB(178,185,244).
		chroma.GenericStrong.Color = hex(lipgloss.Color("#b2b9f4"))
		chroma.GenericStrong.Bold = new(true)
		chroma.GenericEmph.Color = hex(lipgloss.Color("#b2b9f4"))
	}

	// Default response text - brighter white.
	s.Markdown.Document.StylePrimitive.Color = hex(lipgloss.Color("#e8e2d6"))

	// Inline code (backtick spans) - lilac, matching response emphasis.
	s.Markdown.Code.StylePrimitive.Color = hex(lipgloss.Color("#b2b9f4"))

	// User prompt vertical bar - brighter orange.
	s.Messages.UserBlurred = s.Messages.UserBlurred.BorderForeground(lipgloss.Color("#f08060"))
	s.Messages.UserFocused = s.Messages.UserFocused.BorderForeground(lipgloss.Color("#f08060"))

	return s
}

// gruvboxDarkOpts returns the quickStyleOpts for the Gruvbox Dark theme,
// using canonical colors from the morhetz/gruvbox palette.
func gruvboxDarkOpts() quickStyleOpts {
	return quickStyleOpts{
		primary:   lipgloss.Color("#fabd2f"), // yellow
		secondary: lipgloss.Color("#d3869b"), // purple
		accent:    lipgloss.Color("#b8bb26"), // green
		keyword:   lipgloss.Color("#fe8019"), // orange

		fgBase:       lipgloss.Color("#ebdbb2"), // fg
		fgMoreSubtle: lipgloss.Color("#a89984"), // fg4/gray
		fgSubtle:     lipgloss.Color("#bdae93"), // fg3
		fgMostSubtle: lipgloss.Color("#928374"), // gray

		onPrimary: lipgloss.Color("#282828"), // bg on primary

		bgBase:         lipgloss.Color("#282828"), // bg
		bgLeastVisible: lipgloss.Color("#3c3836"), // bg1
		bgLessVisible:  lipgloss.Color("#504945"), // bg2
		bgMostVisible:  lipgloss.Color("#665c54"), // bg3

		separator: lipgloss.Color("#504945"), // bg2

		destructive:       lipgloss.Color("#fb4934"), // red bright
		error:             lipgloss.Color("#cc241d"), // red dark
		warningSubtle:     lipgloss.Color("#fabd2f"), // yellow bright
		warning:           lipgloss.Color("#d79921"), // yellow dark
		attention:         lipgloss.Color("#fe8019"), // orange
		busy:              lipgloss.Color("#fabd2f"), // yellow bright
		info:              lipgloss.Color("#83a598"), // blue bright
		infoMoreSubtle:    lipgloss.Color("#83a598"), // blue bright
		infoMostSubtle:    lipgloss.Color("#458588"), // blue dark
		success:           lipgloss.Color("#b8bb26"), // green bright
		successMoreSubtle: lipgloss.Color("#b8bb26"), // green bright
		successMostSubtle: lipgloss.Color("#8ec07c"), // aqua bright

		// ANSI 16-color palette for remapping raw terminal output
		// (e.g. bang-mode shell commands) onto legible Gruvbox colors.
		ansiBlack:   lipgloss.Color("#282828"),
		ansiRed:     lipgloss.Color("#cc241d"),
		ansiGreen:   lipgloss.Color("#98971a"),
		ansiYellow:  lipgloss.Color("#d79921"),
		ansiBlue:    lipgloss.Color("#458588"),
		ansiMagenta: lipgloss.Color("#b16286"),
		ansiCyan:    lipgloss.Color("#689d6a"),
		ansiWhite:   lipgloss.Color("#a89984"),

		ansiBrightBlack:   lipgloss.Color("#928374"),
		ansiBrightRed:     lipgloss.Color("#fb4934"),
		ansiBrightGreen:   lipgloss.Color("#b8bb26"),
		ansiBrightYellow:  lipgloss.Color("#fabd2f"),
		ansiBrightBlue:    lipgloss.Color("#83a598"),
		ansiBrightMagenta: lipgloss.Color("#d3869b"),
		ansiBrightCyan:    lipgloss.Color("#8ec07c"),
		ansiBrightWhite:   lipgloss.Color("#ebdbb2"),
	}
}

// builtinThemes maps theme names to their quickStyleOpts palette definitions.
var builtinThemes = map[string]func() quickStyleOpts{
	"charmtone":    charmtoneOpts,
	"gruvbox-dark": gruvboxDarkOpts,
	"orange-crush": orangeCrushOpts,
}

// builtinThemeOverrides maps theme names to functions that apply
// theme-specific style tweaks on top of the styles produced by
// [quickStyle]. Themes without overrides are absent from the map.
var builtinThemeOverrides = map[string]func(Styles) Styles{
	"charmtone":    charmtoneOverrides,
	"orange-crush": orangeCrushOverrides,
}

// BuiltinThemeNames returns the names of all built-in themes, sorted.
func BuiltinThemeNames() []string {
	names := make([]string, 0, len(builtinThemes))
	for name := range builtinThemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LoadTheme loads a theme by built-in name. Returns CharmtonePantera styles
// for an empty name. Returns an error if the name is not recognized.
func LoadTheme(name string) (Styles, error) {
	if name == "" {
		return CharmtonePantera(), nil
	}
	key := strings.ToLower(name)
	optsFn, ok := builtinThemes[key]
	if !ok {
		return Styles{}, fmt.Errorf("unknown theme %q; available themes: %s", name, strings.Join(BuiltinThemeNames(), ", "))
	}
	s := quickStyle(optsFn())
	if override, ok := builtinThemeOverrides[key]; ok {
		s = override(s)
	}
	return s, nil
}
