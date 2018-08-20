package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"path"
	"strings"
	"time"
)

func IsImageURL(url string) bool {
	switch strings.ToLower(path.Ext(url)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
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
