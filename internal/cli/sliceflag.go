package cli

import (
	"errors"
	"strings"
)

type stringSliceFlag []string

func (f *stringSliceFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return errors.New("empty value")
		}
		out = append(out, part)
	}
	*f = append(*f, out...)
	return nil
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
