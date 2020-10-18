package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"github.com/auttaja/gommand"
	"github.com/hako/durafmt"
	"time"
)

func init() {
	router.Router.SetCommand(&gommand.Command{
		Name:        "ping",
		Description: "Pings the bot.",
		Category:    categories.Informational,
		Function: func(ctx *gommand.Context) error {
			_, _ = ctx.Reply("🏓 **Pong!**", durafmt.Parse(time.Now().UTC().Sub(ctx.Message.ID.Date())))
			return nil
		},
	})
}
