package models

type Branch struct {
	Name       string
	Hash       string
	IsCurrent  bool
	IsRemote   bool
	Upstream   string // e.g., "origin/main: ahead 2"
	LastCommit string
}
