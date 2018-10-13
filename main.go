package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"sort"

	"github.com/ahmdrz/goinsta"
)

var (
	subreddit = flag.String("sub", "memes", "The Subreddit to pull from")
	username  = flag.String("username", "", "Instagram Username")
	password  = flag.String("password", "", "Instagram Password")
	storedir  = flag.String("store", "used", "Storage directory")
	minscore  = flag.Int("minscore", 100, "Minimum score")
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
	p, err := MakeImagePost(st, unused)
	if err != nil {
		return err
	}
	if err := st.Insert(p.Submission); err != nil {
		return err
	}
	fmt.Println(p)
	if *dryrun {
		return SavePost(p.Image)
	}
	return UploadPost(p)
}

type Post struct {
	Image      image.Image
	Caption    string
	Submission Submission
}

func (p Post) String() string {
	return fmt.Sprintf("Title: %s, Caption: %s", p.Submission.Title, p.Caption)
}

func UploadPost(p *Post) error {
	insta := goinsta.New(*username, *password)
	if err := insta.Login(); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}
	defer insta.Logout()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, p.Image, nil); err != nil {
		return err
	}
	if _, err := insta.UploadPhoto(&buf, p.Caption, 87, 0); err != nil {
		return fmt.Errorf("failed to upload: %v", err)
	}
	return nil
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
