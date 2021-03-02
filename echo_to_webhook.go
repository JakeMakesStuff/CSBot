package main

import (
	"CSBot/db"
	"context"
	"github.com/andersfylling/disgord"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func messageCreateEcho(s disgord.Session, evt *disgord.MessageCreate) {
	rows, err := db.Conn.Query(context.TODO(), "SELECT webhook_url FROM webhook_logging WHERE channel_id = $1", uint64(evt.Message.ChannelID))
	if err != nil {
		log.Fatalln(err)
		return
	}
	for rows.Next() {
		var webhookUrl string
		if err = rows.Scan(&webhookUrl); err != nil {
			log.Println(err)
			continue
		}
		u, err := url.Parse(webhookUrl)
		if err != nil {
			log.Println(err)
			continue
		}
		split := strings.Split(u.Path, "/")
		if len(split) != 5 {
			println("bad URL:", split)
			continue
		}
		ret, err := disgord.NewExecuteWebhookParams(disgord.ParseSnowflakeString(split[3]), split[4])
		if err != nil {
			log.Println(err)
			continue
		}
		ret.Username = evt.Message.Author.Username
		a, _ := evt.Message.Author.AvatarURL(1024, true)
		ret.AvatarURL = a
		ret.Embeds = evt.Message.Embeds
		ret.Content = evt.Message.Content
		attachments := evt.Message.Attachments
		if len(attachments) != 0 {
			resp, err := http.Get(attachments[0].URL)
			if err != nil {
				log.Println(err)
				continue
			}
			ret.File = &disgord.CreateMessageFileParams{
				Reader:     resp.Body,
				FileName:   attachments[0].Filename,
				SpoilerTag: attachments[0].SpoilerTag,
			}
		}
		_, err = s.ExecuteWebhook(context.TODO(), ret, true, "")
		if err != nil {
			log.Println(err)
		}
	}
}
