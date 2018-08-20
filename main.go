package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ahmdrz/goinsta"
	"gopkg.in/jdkato/prose.v2"
)

var (
	subreddit = flag.String("sub", "UnethicalLifeProTips", "The Subreddit to pull from")
	username  = flag.String("username", "unethicallifeprotips", "Instagram Username")
	password  = flag.String("password", "", "Instagram Password")
	storedir  = flag.String("store", "used", "Storage directory")
	minscore  = flag.Int("minscore", 100, "Minimum score")
	imgpost   = flag.Bool("imgpost", false, "Post Image")
	dryrun    = flag.Bool("dry", false, "Don't actually post the image")
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
	seen := map[string]bool{}
	var caption strings.Builder
	for _, tok := range doc.Tokens() {
		if seen[tok.Text] {
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

func IsImageURL(url string) bool {
	switch strings.ToLower(path.Ext(url)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

func FetchImage(url string) (image.Image, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch strings.ToLower(path.Ext(url)) {
	case ".jpg", ".jpeg":
		return jpeg.Decode(resp.Body)
	case ".png":
		return png.Decode(resp.Body)
	default:
		return nil, fmt.Errorf("unsuported image url: %s", url)
	}
}

func MakePost(s Submission) (image.Image, string, error) {
	if *imgpost {
		if !IsImageURL(s.Url) {
			return nil, "", fmt.Errorf("not an image post: %s", s.Url)
		}
		im, err := FetchImage(s.Url)
		if err != nil {
			return nil, "", err
		}
		return im, s.Title, nil
	}
	im, err := MakeImage(s.Title)
	if err != nil {
		return nil, "", err
	}
	cap, err := MakeCaption(s.Title)
	if err != nil {
		return nil, "", err
	}
	return im, cap, nil
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
		im, cap, err := MakePost(s)
		if err != nil {
			log.Println(err)
			continue
		}
		if err := st.Insert(s); err != nil {
			return err
		}
		fmt.Printf("Score: %d\nTitle: %s\nCaption: %s\n", s.Score, s.Title, cap)
		if *dryrun {
			fmt.Println("writing to post.jpeg")
			f, err := os.Create("post.jpeg")
			if err != nil {
				return err
			}
			defer f.Close()
			return jpeg.Encode(f, im, &jpeg.Options{Quality: jpeg.DefaultQuality})
		}
		return PostImage(im, cap)
	}
	return fmt.Errorf("all %d submissions are used", len(ss))
}

func PostImage(m image.Image, caption string) error {
	insta := goinsta.New(*username, *password)
	if err := insta.Login(); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}
	defer insta.Logout()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, m, nil); err != nil {
		return err
	}
	if _, err := insta.UploadPhoto(&buf, caption, 87, 0); err != nil {
		return fmt.Errorf("failed to upload: %v", err)
	}
	return nil
}
