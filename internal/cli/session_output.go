package cli

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/mattn/go-isatty"
)

type sessionOutput struct {
	Version string

	Status string

	ForwardingFrom string
	ForwardingTo   string

	Inspector string
}

func printSession(w io.Writer, out sessionOutput) {
	color := wantsColor(w)
	st := ansiStyle{enabled: color}

	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintf(w, "%s %s\n\n", st.brand("Eosrift"), st.dim(out.Version))

	const labelWidth = 13
	row := func(label, value string) {
		_, _ = fmt.Fprintf(w, "  %s  %s\n", st.dim(fmt.Sprintf("%-*s", labelWidth, label)), value)
	}

	row("Session Status", st.ok(out.Status))
	_, _ = fmt.Fprintln(w, "")

	if out.ForwardingFrom != "" {
		row("Forwarding", fmt.Sprintf("%s %s %s", st.url(out.ForwardingFrom), st.dim("→"), st.dim(out.ForwardingTo)))
	}
	if out.Inspector != "" {
		row("Inspector", st.url(out.Inspector))
	}
}

type ansiStyle struct {
	enabled bool
}

func (s ansiStyle) wrap(code, text string) string {
	if !s.enabled || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (s ansiStyle) brand(text string) string { return s.wrap("35", text) }
func (s ansiStyle) ok(text string) string    { return s.wrap("32", text) }
func (s ansiStyle) url(text string) string   { return s.wrap("94", text) }
func (s ansiStyle) dim(text string) string   { return s.wrap("90", text) }

func wantsColor(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func displayHostPort(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "127.0.0.1" || host == "::1" {
		host = "localhost"
	}
	return host + ":" + port
}
