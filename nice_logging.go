package main

import (
	"CSBot/categories"
	"CSBot/router"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
	"strings"
	"sync"
)

var (
	niceLock    = sync.RWMutex{}
	niceContent = ""
)

func niceLog(text string) {
	niceLock.Lock()
	niceContent = text
	niceLock.Unlock()
}

func init() {
	router.Router.SetCommand(&gommand.Command{
		Name:        "nicelog",
		Description: "Sends the text from the last nice message.",
		Category:    categories.Informational,
		Function: func(ctx *gommand.Context) error {
			niceLock.RLock()
			content := niceContent
			niceLock.RUnlock()
			if content == "" {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Nice hasn't been triggered yet.")
				return nil
			}
			_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Here is the content the bot saw when it responded with nice.", &disgord.CreateMessageFileParams{
				Reader:     strings.NewReader(content),
				FileName:   "content.txt",
				SpoilerTag: false,
			})
			return nil
		},
	})
}
