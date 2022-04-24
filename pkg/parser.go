package pkg

import (
	"log"
	"strings"
	"time"
)

type ParsedText struct {
	Items    []*ListItem
	MetaData *MetaData
}

type ListItem struct {
	Value       string
	URL         string
	Variable    string
	IsURL       bool
	IsBlock     bool
	IsText      bool
	IsHeaderOne bool
	IsHeaderTwo bool
}

type MetaData struct {
	PublishAt   *time.Time
	Title       string
	Description string
}

var urlToken = "=>"
var blockToken = ">"
var varToken = "=@"
var headerOneToken = "#"
var headerTwoToken = "##"

type SplitToken struct {
	Key   string
	Value string
}

func TextToSplitToken(text string) *SplitToken {
	txt := strings.Trim(text, " ")
	token := &SplitToken{}
	word := ""
	for i, c := range txt {
		if c == ' ' {
			token.Key = strings.Trim(word, " ")
			token.Value = strings.Trim(txt[i:], " ")
			break
		} else {
			word += string(c)
		}
	}

	if token.Key == "" {
		token.Key = text
		token.Value = text
	}

	return token
}

func SplitByNewline(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

func ParseText(text string) *ParsedText {
	textItems := SplitByNewline(text)
	items := []*ListItem{}
	meta := &MetaData{}

	for _, t := range textItems {
		li := &ListItem{
			Value: strings.Trim(t, " "),
		}

		if strings.HasPrefix(li.Value, urlToken) {
			li.IsURL = true
			split := TextToSplitToken(strings.Replace(li.Value, urlToken, "", 1))
			li.URL = split.Key
			if split.Value == "" {
				li.Value = split.Key
			} else {
				li.Value = split.Value
			}
		} else if strings.HasPrefix(li.Value, blockToken) {
			li.IsBlock = true
			li.Value = strings.Replace(li.Value, blockToken, "", 1)
		} else if strings.HasPrefix(li.Value, varToken) {
			split := TextToSplitToken(strings.Replace(li.Value, varToken, "", 1))
			if split.Key == "publish_at" {
				date, err := time.Parse("2006-02-15", split.Value)
				if err != nil {
					log.Println(err)
				}
				meta.PublishAt = &date
			}

			if split.Key == "title" {
				meta.Title = split.Value
			}

			if split.Key == "description" {
				meta.Description = split.Value
			}
			continue
		} else if strings.HasPrefix(li.Value, headerTwoToken) {
			li.IsHeaderTwo = true
			li.Value = strings.Replace(li.Value, headerTwoToken, "", 1)
		} else if strings.HasPrefix(li.Value, headerOneToken) {
			li.IsHeaderOne = true
			li.Value = strings.Replace(li.Value, headerOneToken, "", 1)
		} else {
			li.IsText = true
		}

		if len(items) > 0 {
			prevItem := items[len(items)-1]
			if li.Value == "" && prevItem.Value == "" {
				continue
			}
		}

		items = append(items, li)
	}

	if len(items) > 0 {
		last := items[len(items)-1]
		if last.Value == "" {
			items = items[:len(items)-1]
		}
	}

	return &ParsedText{
		Items:    items,
		MetaData: meta,
	}
}
