package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/ahmdrz/goinsta"
	"image"
	"image/jpeg"
	"log"
	"math/rand"
	"sort"
	"time"
)

var (
	subreddit = flag.String("sub", "UnethicalLifeProTips", "The Subreddit to pull from")
	username  = flag.String("username", "unethicallifeprotips", "Instagram Username")
	password  = flag.String("password", "", "Instagram Password")
	caption   = flag.String("caption", "#LPT", "The post caption")
	storedir  = flag.String("store", "used", "Storage directory")
	minscore  = flag.Int("minscore", 100, "Minimum score")
)

func init() {
	rand.Seed(time.Now().Unix())
	flag.Parse()
}

func main() {
	if err := DoPost(); err != nil {
		log.Fatal(err)
	}
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
		if err := st.Insert(s); err != nil {
			return err
		}
		fmt.Printf("Score %d:\n\n%s\n", s.Score, s.Title)
		return PostImage(im)
	}
	return fmt.Errorf("all %d submissions are used", len(ss))
}

func PostImage(m image.Image) error {
	insta := goinsta.New(*username, *password)
	if err := insta.Login(); err != nil {
		return fmt.Errorf("failed to login: %s", err)
	}
	defer insta.Logout()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, m, nil); err != nil {
		return err
	}
	if _, err := insta.UploadPhoto(&buf, *caption, 87, 0); err != nil {
		return fmt.Errorf("failed to upload:", err)
	}
	return nil
}
