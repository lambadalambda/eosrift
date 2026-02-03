package cli

import (
	"flag"
	"strconv"
	"strings"
)

type boolFlag interface {
	IsBoolFlag() bool
}

// parseInterspersedFlags parses flags even if they appear after positional args.
//
// Example: ["8080", "--server", "https://example.com"] becomes
// ["--server", "https://example.com", "8080"] before parsing.
//
// This is closer to typical CLI behavior (including ngrok) than the standard
// flag.FlagSet.Parse, which stops scanning at the first non-flag argument.
func parseInterspersedFlags(fs *flag.FlagSet, args []string) error {
	return fs.Parse(reorderInterspersedFlags(fs, args))
}

func reorderInterspersedFlags(fs *flag.FlagSet, args []string) []string {
	var flags []string
	var positionals []string

	for i := 0; i < len(args); i++ {
		a := args[i]

		if a == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}

		// Treat any -flag or --flag as a flag token (unknown flags will still be
		// handled by fs.Parse).
		if isFlagToken(a) {
			flags = append(flags, a)

			// If the flag is known and expects a value, consume the next token.
			name, hasInlineValue := splitFlagName(a)
			if hasInlineValue {
				continue
			}

			f := fs.Lookup(name)
			if f == nil {
				continue
			}

			// If the flag is boolean, only consume a value if the next token looks
			// like a boolean literal.
			if bf, ok := f.Value.(boolFlag); ok && bf.IsBoolFlag() {
				if i+1 < len(args) && isBoolLiteral(args[i+1]) {
					// The standard flag parser does not accept "-flag false" for
					// boolean flags; it only accepts "-flag=false". Normalize.
					flags[len(flags)-1] = flags[len(flags)-1] + "=" + args[i+1]
					i++
				}
				continue
			}

			// Non-bool flags expect a value token (if present).
			if i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}

			continue
		}

		positionals = append(positionals, a)
	}

	out := make([]string, 0, len(flags)+len(positionals))
	out = append(out, flags...)
	out = append(out, positionals...)
	return out
}

func isFlagToken(s string) bool {
	if s == "" || s == "-" {
		return false
	}
	return strings.HasPrefix(s, "-")
}

func splitFlagName(tok string) (string, bool) {
	trimmed := strings.TrimLeft(tok, "-")
	if trimmed == "" {
		return "", false
	}
	name, _, hasValue := strings.Cut(trimmed, "=")
	return name, hasValue
}

func isBoolLiteral(s string) bool {
	_, err := strconv.ParseBool(s)
	return err == nil
}
