package main

import (
	"github.com/peterbourgon/diskv"
)

type Store struct {
	kv *diskv.Diskv
}

func NewStore(dir string) *Store {
	return &Store{
		kv: diskv.New(diskv.Options{
			BasePath:     dir,
			CacheSizeMax: 1024 * 1024,
		}),
	}
}

func (s *Store) Contains(sub Submission) bool {
	return s.kv.Has(sub.Id)
}

func (s *Store) Insert(sub Submission) error {
	return s.kv.WriteString(sub.Id, sub.Title)
}
