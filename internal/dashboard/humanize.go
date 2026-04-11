package dashboard

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func FormatCount(value int) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	raw := fmt.Sprintf("%d", value)
	var out []byte
	for i, r := range raw {
		if i > 0 && (len(raw)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(r))
	}
	return sign + string(out)
}

func CompactSessionID(sessionID string) string {
	if len(sessionID) <= 10 {
		if strings.TrimSpace(sessionID) == "" {
			return "n/a"
		}
		return sessionID
	}
	return sessionID[:4] + "..." + sessionID[len(sessionID)-6:]
}

func SanitizeInline(value string) string {
	replacer := strings.NewReplacer("\\r\\n", " ", "\\r", " ", "\\n", " ", "\r\n", " ", "\r", " ", "\n", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func Truncate(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

func formatRuntime(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
}

func formatRuntimeTurns(seconds, turns int) string {
	if turns > 0 {
		return fmt.Sprintf("%s / %d", formatRuntime(seconds), turns)
	}
	return formatRuntime(seconds)
}

func formatDue(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	ms := int(math.Ceil(float64(duration) / float64(time.Millisecond)))
	return fmt.Sprintf("%d.%03ds", ms/1000, ms%1000)
}
