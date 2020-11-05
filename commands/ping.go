package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"context"
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
			t1 := time.Now().UTC()
			msg, err := ctx.Reply("Pinging...")
			if err != nil {
				return nil
			}
			_, _ = msg.Reply(context.TODO(), ctx.Session, "🏓 **Pong!**", durafmt.Parse(time.Now().UTC().Sub(t1)))
			return nil
		},
	})
}
