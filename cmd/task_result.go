package cmd

import (
	"fmt"

	"github.com/dyatlov/go-opengraph/opengraph"
)

type TaskResult struct {
	URL         string
	Title       string
	Description string
	Image       string
	Og          *opengraph.OpenGraph
	Err         error
}

func (tr *TaskResult) String() string {
	return fmt.Sprintf(
		"TaskResult: {url:%s, title:%s, description:%s, image:%s, og:%v, err:%v}",
		tr.URL,
		tr.Title,
		tr.Description,
		tr.Image,
		tr.Og,
		tr.Err,
	)
}
