package main

import (
	"CSBot/categories"
	_ "CSBot/commands"
	"CSBot/router"
	"context"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
	"os"
)

func main() {
	s, err := disgord.NewClient(disgord.Config{
		BotToken: os.Getenv("TOKEN"),
	})
	if err != nil {
		panic(err)
	}
	s.On(disgord.EvtMessageCreate, messageCreateChair)
	s.On(disgord.EvtMessageCreate, messageCreateNice)
	s.On(disgord.EvtMessageCreate, messageCreateEcho)
	router.Router.Hook(s)
	router.Router.GetCommand("help").(*gommand.Command).Category = categories.Informational
	err = s.StayConnectedUntilInterrupted(context.TODO())
	if err != nil {
		panic(err)
	}
}
