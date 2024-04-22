package repository

import (
	"strings"
)

type CommitMessage struct {
	Headline string
	Body     string
}

func (c CommitMessage) String() string {
	commitMsg := new(strings.Builder)
	commitMsg.WriteString(c.Headline)
	if len(c.Body) > 0 {
		commitMsg.WriteString("\n\n")
		commitMsg.WriteString(c.Body)
	}
	return commitMsg.String()
}

func NewCommitMessage(title, body, footer string) CommitMessage {
	bodyWithFooter := new(strings.Builder)
	if len(body) > 0 {
		bodyWithFooter.WriteString(body)
	}
	if len(footer) > 0 {
		if bodyWithFooter.Len() > 0 {
			bodyWithFooter.WriteString("\n\n")
		}
		bodyWithFooter.WriteString("-- \n")
		bodyWithFooter.WriteString(footer)
	}

	return CommitMessage{
		Headline: title,
		Body:     bodyWithFooter.String(),
	}
}
