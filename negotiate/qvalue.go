package negotiate

import (
	"slices"
	"strconv"
	"strings"
)

func SelectQValue(header string, allowed []string) string {
	formats := strings.Split(header, ",")
	best := ""
	bestQ := 0.0
	for _, format := range formats {
		parts := strings.Split(format, ";")
		name := strings.Trim(parts[0], " \t")

		if !slices.Contains(allowed, name) {
			continue
		}

		q := 1.0
		if len(parts) > 1 {
			trimmed := strings.Trim(parts[1], " \t")
			if strings.HasPrefix(trimmed, "q=") {
				q, _ = strconv.ParseFloat(trimmed[2:], 64)
			}
		}

		if q > bestQ || (q == bestQ && name == allowed[0]) {
			bestQ = q
			best = name
		}
	}

	return best
}

func SelectQValueFast(header string, allowed []string) string {
	best := ""
	bestQ := 0.0

	name := ""
	start := 0
	end := 0

	for pos, char := range header {
		if char == ';' {
			name = header[start : end+1]
			start = pos + 1
			end = start
			continue
		}

		if char == ',' || pos == len(header)-1 {
			q := 1.0
			if char != ',' && char != ' ' && char != '\t' {
				end = pos
			}
			if name == "" {
				name = header[start : end+1]
			} else if len(header) > end+1 {
				if parsed, _ := strconv.ParseFloat(header[start+2:end+1], 64); parsed > 0 {
					q = parsed
				}
			}
			start = pos + 1
			end = start

			if !slices.Contains(allowed, name) {
				name = ""
				continue
			}

			if q > bestQ || (q == bestQ && name == allowed[0]) {
				bestQ = q
				best = name
			}
			name = ""
			continue
		}

		if char != ' ' && char != '\t' {
			end = pos
			if header[start] == ' ' || header[start] == '\t' {
				start = pos
			}
		}
	}

	return best
}
