package time

import "strings"

func FormatTime(format string) string {
	mapping := map[string]string{
		// Year
		"Y": "2006", // 4 digit year (e.g., 2006)
		"y": "06",   // 2 digit year (e.g., 06)

		// Month
		"F": "January", // Full month name (e.g., January)
		"M": "Jan",     // 3 letter month name (e.g., Jan)
		"m": "01",      // Month with leading zeros (01-12)
		"n": "1",       // Month without leading zeros (1-12)

		// Day
		"d": "02",     // Day of the month with leading zeros (01-31)
		"j": "2",      // Day of the month without leading zeros (1-31)
		"D": "Mon",    // 3 letter day name (e.g., Mon)
		"l": "Monday", // Full day name (e.g., Monday)

		// Time
		"H": "15", // 24-hour format with leading zeros (00-23)
		"G": "15", // 24-hour format without leading zeros (0-23)
		"h": "03", // 12-hour format with leading zeros (01-12)
		"g": "3",  // 12-hour format without leading zeros (1-12)
		"i": "04", // Minutes with leading zeros (00-59)
		"s": "05", // Seconds with leading zeros (00-59)
		"A": "PM", // AM/PM
		"a": "pm", // am/pm

		// Others
		"T": "MST",                             // Timezone abbreviation
		"Z": "-0700",                           // Timezone offset
		"c": "2006-01-02T15:04:05-07:00",       // ISO 8601
		"r": "Mon, 02 Jan 2006 15:04:05 -0700", // RFC 2822
	}

	var result strings.Builder

	for i := 0; i < len(format); i++ {
		char := string(format[i])

		// If we find a character to replace, use the mapping
		if replacement, exists := mapping[char]; exists {
			result.WriteString(replacement)
		} else {
			// Handle escaping
			if char == "\\" && i+1 < len(format) {
				i++
				result.WriteString(string(format[i]))
			} else {
				result.WriteString(char)
			}
		}
	}

	return result.String()
}
