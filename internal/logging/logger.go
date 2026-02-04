package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

func ParseLevel(s string) (Level, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug, true
	case "info", "":
		return LevelInfo, true
	case "warn", "warning":
		return LevelWarn, true
	case "error":
		return LevelError, true
	default:
		return LevelInfo, false
	}
}

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

func ParseFormat(s string) (Format, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "text", "":
		return FormatText, true
	case "json":
		return FormatJSON, true
	default:
		return FormatText, false
	}
}

type Field struct {
	Key   string
	Value any
}

func F(key string, value any) Field {
	return Field{Key: strings.TrimSpace(key), Value: value}
}

type Options struct {
	Out    io.Writer
	Level  Level
	Format Format
	Now    func() time.Time
}

type Logger interface {
	With(fields ...Field) Logger

	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

type logger struct {
	out    io.Writer
	level  Level
	format Format
	now    func() time.Time
	base   []Field
	mu     *sync.Mutex
}

func New(opts Options) Logger {
	out := opts.Out
	if out == nil {
		out = os.Stderr
	}

	format := opts.Format
	if format == "" {
		format = FormatText
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	return &logger{
		out:    out,
		level:  opts.Level,
		format: format,
		now:    now,
		mu:     &sync.Mutex{},
	}
}

func (l *logger) With(fields ...Field) Logger {
	if l == nil {
		return New(Options{})
	}

	base := append([]Field{}, l.base...)
	base = append(base, fields...)

	return &logger{
		out:    l.out,
		level:  l.level,
		format: l.format,
		now:    l.now,
		base:   base,
		mu:     l.mu,
	}
}

func (l *logger) Debug(msg string, fields ...Field) { l.log(LevelDebug, msg, fields...) }
func (l *logger) Info(msg string, fields ...Field)  { l.log(LevelInfo, msg, fields...) }
func (l *logger) Warn(msg string, fields ...Field)  { l.log(LevelWarn, msg, fields...) }
func (l *logger) Error(msg string, fields ...Field) { l.log(LevelError, msg, fields...) }

func (l *logger) log(level Level, msg string, fields ...Field) {
	if l == nil {
		return
	}
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	ts := l.now().UTC().Format(time.RFC3339Nano)

	switch l.format {
	case FormatJSON:
		m := make(map[string]any, 3+len(l.base)+len(fields))
		m["ts"] = ts
		m["level"] = level.String()
		m["msg"] = msg

		for _, f := range l.base {
			addJSONField(m, f)
		}
		for _, f := range fields {
			addJSONField(m, f)
		}

		enc := json.NewEncoder(l.out)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(m)

	default:
		var sb strings.Builder
		sb.WriteString(ts)
		sb.WriteString(" level=")
		sb.WriteString(level.String())
		sb.WriteString(" msg=")
		sb.WriteString(strconv.Quote(msg))

		all := make([]Field, 0, len(l.base)+len(fields))
		all = append(all, l.base...)
		all = append(all, fields...)

		sort.SliceStable(all, func(i, j int) bool { return all[i].Key < all[j].Key })

		for _, f := range all {
			key := strings.TrimSpace(f.Key)
			if key == "" {
				continue
			}
			sb.WriteString(" ")
			sb.WriteString(key)
			sb.WriteString("=")
			sb.WriteString(formatTextValue(f.Value))
		}

		sb.WriteString("\n")
		_, _ = io.WriteString(l.out, sb.String())
	}
}

func addJSONField(m map[string]any, f Field) {
	key := strings.TrimSpace(f.Key)
	if key == "" {
		return
	}
	m[key] = jsonValue(f.Value)
}

func jsonValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case error:
		return x.Error()
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	case []byte:
		return string(x)
	default:
		if _, err := json.Marshal(x); err != nil {
			return fmt.Sprint(x)
		}
		return x
	}
}

func formatTextValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case error:
		return strconv.Quote(x.Error())
	case string:
		return strconv.Quote(x)
	case []byte:
		return strconv.Quote(string(x))
	case time.Time:
		return strconv.Quote(x.UTC().Format(time.RFC3339Nano))
	default:
		s := fmt.Sprint(x)
		// Quote if it contains whitespace to keep it parseable.
		if strings.IndexFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }) >= 0 {
			return strconv.Quote(s)
		}
		return s
	}
}

