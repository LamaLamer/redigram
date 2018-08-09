package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"sort"
	"strings"

	"github.com/ahmdrz/goinsta"
	"gopkg.in/jdkato/prose.v2"
)

var (
	subreddit = flag.String("sub", "UnethicalLifeProTips", "The Subreddit to pull from")
	username  = flag.String("username", "unethicallifeprotips", "Instagram Username")
	password  = flag.String("password", "", "Instagram Password")
	storedir  = flag.String("store", "used", "Storage directory")
	minscore  = flag.Int("minscore", 100, "Minimum score")
)

func init() {
	flag.Parse()
}

func main() {
	if err := DoPost(); err != nil {
		log.Fatal(err)
	}
}

const MaxCaptionLen = 2000

func MakeCaption(s string) (string, error) {
	doc, err := prose.NewDocument(s)
	if err != nil {
		return "", err
	}
	var caption strings.Builder
	for _, tok := range doc.Tokens() {
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

func DoPost() error {
	st := NewStore(*storedir)
	ss, err := GetSubmissions(*subreddit)
	if err != nil {
		return err
	}
	sort.Sort(ByScore(ss))
	for _, s := range ss {
		if st.Contains(s) || s.Score < *minscore {
			continue
		}
		im, err := MakeImage(s.Title)
		if err != nil {
			return err
		}
		cap, err := MakeCaption(s.Title)
		if err != nil {
			return err
		}
		if err := st.Insert(s); err != nil {
			return err
		}
		fmt.Printf("Score %d:\n\n%s\n", s.Score, s.Title)
		return PostImage(im, cap)
	}
	return fmt.Errorf("all %d submissions are used", len(ss))
}

func PostImage(m image.Image, caption string) error {
	insta := goinsta.New(*username, *password)
	if err := insta.Login(); err != nil {
		return fmt.Errorf("failed to login: %s", err)
	}
	defer insta.Logout()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, m, nil); err != nil {
		return err
	}
	if _, err := insta.UploadPhoto(&buf, caption, 87, 0); err != nil {
		return fmt.Errorf("failed to upload:", err)
	}
	return nil
}
