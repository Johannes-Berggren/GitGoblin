package models

import "time"

type Commit struct {
	Hash      string
	ShortHash string
	Author    string
	Email     string
	Date      time.Time
	Message   string
	Refs      []string // branch names, tags
	Parents   []string
}
