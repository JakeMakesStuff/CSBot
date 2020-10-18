package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
	"runtime"
	"strconv"
)

func init() {
	router.Router.SetCommand(&gommand.Command{
		Name:        "botinfo",
		Description: "Sends information about the bot.",
		Category:    categories.Informational,
		Function: func(ctx *gommand.Context) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			_, _ = ctx.Reply(&disgord.Embed{
				Description: "**CSBot Information:**",
				Fields: []*disgord.EmbedField{
					{
						Name: "Go Version:",
						Value: runtime.Version(),
						Inline: true,
					},
					{
						Name: "Disgord Version:",
						Value: disgord.Version,
						Inline: true,
					},
					{
						Name: "Gommand Version:",
						Value: "v1.11.1",
						Inline: true,
					},
					{
						Name: "Running Goroutines:",
						Value: strconv.Itoa(runtime.NumGoroutine()),
						Inline: true,
					},
					{
						Name: "RAM Usage:",
						Value: strconv.Itoa(int(m.Alloc/1000000))+"MB",
						Inline: true,
					},
					{
						Name: "Garbage Collections:",
						Value: strconv.Itoa(int(m.NumGC)),
						Inline: true,
					},
				},
			})
			return nil
		},
	})
}
