package store

import (
	"errors"
	"strings"
)

func stringsLower(s string) string    { return strings.ToLower(strings.TrimSpace(s)) }
func hasPrefix(s, prefix string) bool { return strings.HasPrefix(s, prefix) }
func contains(s, part string) bool    { return strings.Contains(s, part) }
func replaceSlash(s string) string    { return strings.ReplaceAll(s, `\`, `/`) }
func errString(msg string) error      { return errors.New(msg) }
