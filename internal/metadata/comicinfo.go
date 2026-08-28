package metadata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func PatchComicInfo(data []byte, storyArc string, rank int) ([]byte, error) {
	if err := validateComicInfo(data); err != nil {
		return nil, err
	}

	text := string(data)
	for _, name := range []string{"AlternateSeries", "AlternateNumber", "AlternateCount"} {
		text = removeElement(text, name)
	}

	var found bool
	text, found = replaceElementText(text, "StoryArc", storyArc)
	if !found {
		var err error
		text, err = appendElement(text, "StoryArc", storyArc)
		if err != nil {
			return nil, err
		}
	}

	text, found = replaceElementText(text, "StoryArcNumber", strconv.Itoa(rank))
	if !found {
		var err error
		text, err = appendElement(text, "StoryArcNumber", strconv.Itoa(rank))
		if err != nil {
			return nil, err
		}
	}

	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return []byte(text), nil
}

func validateComicInfo(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	foundRoot := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "ComicInfo" {
			return fmt.Errorf("expected ComicInfo root, got %s", start.Name.Local)
		}
		foundRoot = true
		break
	}
	if !foundRoot {
		return fmt.Errorf("missing ComicInfo root")
	}
	return nil
}

func replaceElementText(text, name, value string) (string, bool) {
	pattern := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(name) + `(?:\s[^>]*)?>.*?</` + regexp.QuoteMeta(name) + `>`)
	location := pattern.FindStringIndex(text)
	if location == nil {
		return text, false
	}

	segment := text[location[0]:location[1]]
	startEnd := strings.IndexByte(segment, '>')
	endStart := strings.LastIndex(segment, "</"+name+">")
	if startEnd == -1 || endStart == -1 || startEnd >= endStart {
		return text, false
	}

	patched := segment[:startEnd+1] + escapeText(value) + segment[endStart:]
	return text[:location[0]] + patched + text[location[1]:], true
}

func removeElement(text, name string) string {
	quoted := regexp.QuoteMeta(name)
	pattern := regexp.MustCompile(`(?s)\s*<` + quoted + `(?:\s[^>]*)?(?:/>|>.*?</` + quoted + `>)`)
	return pattern.ReplaceAllString(text, "")
}

func appendElement(text, name, value string) (string, error) {
	index := strings.LastIndex(text, "</ComicInfo>")
	if index == -1 {
		return "", fmt.Errorf("missing ComicInfo closing tag")
	}

	insert := fmt.Sprintf("  <%s>%s</%s>\n", name, escapeText(value), name)
	if index > 0 && text[index-1] != '\n' {
		insert = "\n" + insert
	}
	return text[:index] + insert + text[index:], nil
}

func escapeText(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}
