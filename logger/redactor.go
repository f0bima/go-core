package logger

import (
	"strings"
	"log/slog"
)

var sensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"secret":        true,
	"authorization": true,
}

func redactAny(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		redacted := make(map[string]interface{})
		for k, val := range v {
			if sensitiveKeys[strings.ToLower(k)] {
				redacted[k] = "[REDACTED]"
			} else {
				redacted[k] = redactAny(val)
			}
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(v))
		for i, val := range v {
			redacted[i] = redactAny(val)
		}
		return redacted
	default:
		return v
	}
}

func redactAttr(a slog.Attr) slog.Attr {
	if sensitiveKeys[strings.ToLower(a.Key)] {
		return slog.String(a.Key, "[REDACTED]")
	}

	if a.Value.Kind() == slog.KindAny {
		return slog.Any(a.Key, redactAny(a.Value.Any()))
	}

	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		redactedAttrs := make([]any, len(attrs))
		for i, attr := range attrs {
			redactedAttrs[i] = redactAttr(attr)
		}
		return slog.Group(a.Key, redactedAttrs...)
	}

	return a
}
