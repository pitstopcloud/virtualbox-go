package virtualbox

import (
	"bufio"
	"regexp"
	"strings"
)

func parseKeyValues(text string, regexp *regexp.Regexp, callback func(key, val string) error) error {
	return tryParseKeyValues(text, regexp, func(key, val string, ok bool) error {
		if ok {
			return callback(key, val)
		}
		return nil
	})
}

func tryParseKeyValues(stdOut string, regexp *regexp.Regexp, callback func(key, val string, ok bool) error) error {
	r := strings.NewReader(stdOut)
	s := bufio.NewScanner(r)

	for s.Scan() {
		line := s.Text()

		if strings.TrimSpace(line) == "" {
			callback("", "", false)
			continue
		}

		res := regexp.FindStringSubmatch(line)
		if res == nil {
			callback("", line, false)
			continue
		}

		key, val := res[1], res[2]
		if err := callback(key, val, true); err != nil {
			return err
		}
	}

	return s.Err()
}
