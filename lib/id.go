package lib

import (
	"regexp"
	"strings"
)

func ToIdiomID(idiom string) string {
	matcher, _ := regexp.Compile("[^0-9a-zA-Z]+")
	tokens := matcher.Split(idiom, -1)
	lowered := []string{}

	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		lowered = append(lowered, strings.ToLower(token))
	}

	return strings.Join(lowered, "-")
}
