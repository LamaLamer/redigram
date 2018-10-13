package main

import (
	"fmt"
	"strings"

	"gopkg.in/jdkato/prose.v2"
)

const MaxCaptionLen = 2000

func MakeCaption(s string) (string, error) {
	doc, err := prose.NewDocument(s)
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	var caption strings.Builder
	for _, tok := range doc.Tokens() {
		if len(tok.Text) < 2 || seen[tok.Text] {
			continue
		}
		seen[tok.Text] = true
		if caption.Len()+len(tok.Text) > MaxCaptionLen {
			break
		}
		switch tok.Tag {
		case "NNP", "NN", "JJ":
			fmt.Fprintf(&caption, "#%s ", tok.Text)
		}
	}
	return caption.String(), nil
}
