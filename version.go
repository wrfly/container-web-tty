package main

import (
	"github.com/urfave/cli/v2"
)

var (
	Version  string
	CommitID string
	BuildAt  string
)

var author = []*cli.Author{
	&cli.Author{
		Name:  "wrfly",
		Email: "mr.wrfly@gmail.com",
	},
}
