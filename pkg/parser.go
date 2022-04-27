package pkg

import (
	"errors"
	"strconv"
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
	IsImg       bool
}

type MetaData struct {
	PublishAt   *time.Time
	Title       string
	Description string
	ListType    string // https://developer.mozilla.org/en-US/docs/Web/CSS/list-style-type
}

var urlToken = "=>"
var blockToken = ">"
var varToken = "=:"
var imgToken = "=<"
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

func PublishAtDate(date string) (*time.Time, error) {
	e := errors.New("Date must be in this format: YYYY-MM-DD")
	sp := strings.Split(date, "-")
	if len(sp) < 3 {
		return nil, e
	}

	year, err := strconv.Atoi(sp[0])
	if err != nil {
		return nil, e
	}

	m, err := strconv.Atoi(sp[1])
	if err != nil {
		return nil, e
	}

	month := time.Month(m)
	day, err := strconv.Atoi(sp[2])
	if err != nil {
		return nil, e
	}

	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &d, nil
}

func ParseText(text string) *ParsedText {
	textItems := SplitByNewline(text)
	items := []*ListItem{}
	meta := &MetaData{
		ListType: "disc",
	}

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
		} else if strings.HasPrefix(li.Value, imgToken) {
			li.IsImg = true
			split := TextToSplitToken(strings.Replace(li.Value, imgToken, "", 1))
			li.URL = split.Key
			if split.Value == "" {
				li.Value = split.Key
			} else {
				li.Value = split.Value
			}
		} else if strings.HasPrefix(li.Value, varToken) {
			split := TextToSplitToken(strings.Replace(li.Value, varToken, "", 1))
			if split.Key == "publish_at" {
				publishAt, err := PublishAtDate(split.Value)
				if err == nil {
					meta.PublishAt = publishAt
				}
			}

			if split.Key == "title" {
				meta.Title = split.Value
			}

			if split.Key == "description" {
				meta.Description = split.Value
			}

			if split.Key == "list_type" {
				meta.ListType = split.Value
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
