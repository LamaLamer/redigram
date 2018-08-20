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

func DoPost() error {
	st := NewStore(*storedir)
	ss, err := FetchSubmissions(*subreddit)
	if err != nil {
		return err
	}
	sort.Sort(ByScore(ss))
	var unused []Submission
	for _, s := range ss {
		if !st.Contains(s) && s.Score >= *minscore {
			unused = append(unused, s)
		}
	}
	p, err := MakePost(st, unused)
	if err != nil {
		return err
	}
	if err := st.Insert(p.Submission); err != nil {
		return err
	}
	if *dryrun {
		return SavePost(p.Image)
	}
	fmt.Println(p)
	return PostImage(p.Image, p.Caption)
}

func MakePost(st *Store, ss []Submission) (*Post, error) {
	if *imgpost {
		return MakeImagePost(st, ss)
	}
	return MakeTextPost(st, ss)
}

func SavePost(im image.Image) error {
	fmt.Println("writing to post.jpeg")
	f, err := os.Create("post.jpeg")
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, im, &jpeg.Options{Quality: jpeg.DefaultQuality})
}

type Post struct {
	Image      image.Image
	Caption    string
	Submission Submission
}

func (p Post) String() string {
	return fmt.Sprintf("Title: %s, Caption: %s", p.Submission.Title, p.Caption)
}

func MakeTextPost(st *Store, ss []Submission) (*Post, error) {
	for _, s := range ss {
		im, err := MakeImage(s.Title)
		if err != nil {
			return nil, err
		}
		cap, err := MakeCaption(s.Title)
		if err != nil {
			return nil, err
		}
		return &Post{
			Image:      im,
			Caption:    cap,
			Submission: s,
		}, nil
	}
	return nil, fmt.Errorf("all %d submissions are used", len(ss))
}

func MakeImagePost(st *Store, ss []Submission) (*Post, error) {
	for _, s := range ss {
		if !IsImageURL(s.Url) {
			continue
		}
		im, err := FetchImage(s.Url)
		if err != nil {
			return nil, err
		}
		return &Post{
			Image:      im,
			Caption:    s.Title,
			Submission: s,
		}, nil
	}
	return nil, fmt.Errorf("all %d submissions are used", len(ss))
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
