package facebook

import "regexp"

var secretSanitizers = []struct {
	re          *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)(access_token=)[^&\s"']+`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?i)(client_secret=)[^&\s"']+`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?i)("access_token"\s*:\s*")[^"]*"`), `${1}[REDACTED]"`},
	{regexp.MustCompile(`(?i)("client_secret"\s*:\s*")[^"]*"`), `${1}[REDACTED]"`},
	{regexp.MustCompile(`(?i)OAuth\s+[^\s"']+`), `OAuth [REDACTED]`},
}

func sanitizeSecrets(s string) string {
	for _, sanitizer := range secretSanitizers {
		s = sanitizer.re.ReplaceAllString(s, sanitizer.replacement)
	}
	return s
}

func safeErrorString(err error) string {
	if err == nil {
		return ""
	}
	return sanitizeSecrets(err.Error())
}
