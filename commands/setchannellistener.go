package commands

import (
	"CSBot/categories"
	"CSBot/db"
	"CSBot/router"
	"context"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
)

func init() {
	router.Router.SetCommand(&gommand.Command{
		Name:        "setchannellistener",
		Description: "Sets a channel listener.",
		Usage:       "<channel ID> <webhook URL>",
		ArgTransformers: []gommand.ArgTransformer{
			{
				Function: gommand.ChannelTransformer,
			},
			{
				Function: gommand.StringTransformer,
			},
		},
		PermissionValidators: []gommand.PermissionValidator{
			func(ctx *gommand.Context) (string, bool) {
				if ctx.Message.Author.ID == 280610586159611905 {
					return "", true
				}
				return "This command is for maintainers only.", false
			},
		},
		Category: categories.Informational,
		Function: func(ctx *gommand.Context) error {
			_, err := db.Conn.Exec(context.TODO(), "INSERT INTO webhook_logging (channel_id, webhook_url) VALUES ($1, $2)",
				ctx.Args[0].(*disgord.Channel).ID, ctx.Args[1].(string))
			if err != nil {
				_, _ = ctx.Reply("Failed to insert:", err.Error())
				return nil
			}
			_, _ = ctx.Reply("Webhook inserted.")
			return nil
		},
	})
}
