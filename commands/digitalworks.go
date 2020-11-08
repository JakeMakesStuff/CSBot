package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"net"
	"strconv"
)

func init() {
	cli, err := client.NewClient("unix:///var/run/docker.sock", "", nil, nil)
	if err != nil {
		fmt.Println("Unable to connect to Docker:", err.Error())
	}

	router.Router.SetCommand(&gommand.Command{
		Name:                 "digitalworks",
		Description:          "Start a sandboxed DigitalWorks VNC environment. You will then get DM'd the VNC hostname and password.",
		Usage: "<width> <height>",
		Category:             categories.Learning,
		ArgTransformers: []gommand.ArgTransformer{
			{
				Function: gommand.UIntTransformer,
			},
			{
				Function: gommand.UIntTransformer,
			},
		},
		Function: func(ctx *gommand.Context) error {
			// TODO: Add width and height support.
			// Get the width and height.
			width, _ := ctx.Args[0].(uint64)
			if width == 0 {
				width = 800
			}
			height, _ := ctx.Args[0].(uint64)
			if height == 0 {
				height = 600
			}

			// Handle if not configured.
			if cli == nil {
				_, _ = ctx.Reply("Docker is not configured.")
				return nil
			}

			// Send the initial embed for starting the container.
			msg, err := ctx.Reply(&disgord.Embed{
				Title: "Creating Docker Container...",
				Description: "Creating a Docker container which contains your DigitalWorks environment.",
			})
			if err != nil {
				return nil
			}

			// Create the container.
			password := uuid.New().String()
			env := []string{"VNC_PASSWORD="+password}
			max := 12999
			min := 12000
			port := strconv.Itoa(rand.Intn(max - min) + min)
			portMap := nat.PortMap{
				"5900/tcp": {
					{
						// TODO: We should probably make this less random, but realistically it's probably fine for now.
						HostPort: port,
					},
				},
			}
			res, err := cli.ContainerCreate(context.TODO(), &container.Config{
				Image: "wine-digitalworks-vnc",
				Env: env,
			}, &container.HostConfig{PortBindings: portMap, PublishAllPorts: true}, nil, msg.Author.ID.String() + "-vnc")
			if err != nil {
				_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
					Title: "Failed to launch container.",
					Description: err.Error(),
				}).Execute()
				return nil
			}
			err = cli.ContainerStart(context.TODO(), res.ID, types.ContainerStartOptions{})
			if err != nil {
				_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
					Title: "Failed to start container.",
					Description: err.Error(),
				}).Execute()
				return nil
			}

			// Delete the old message.
			_ = ctx.Session.DeleteMessage(context.TODO(), msg.ChannelID, msg.ID)

			// Launch eval mode.
			_, _ = ctx.Reply("DigitalWorks environment is now created. To use it, simply connect via VNC to the hostname and password specified in DM's. If you have DM's off, the bot will be unable to DM you, simply run the command again to get this.")
			conn, err := net.Dial("udp", "8.8.8.8:80")
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			remoteAddr := conn.RemoteAddr().(*net.UDPAddr).IP.String()
			_, _, _ = ctx.Message.Author.SendMsg(context.TODO(), ctx.Session, &disgord.Message{Content: "Hostname: "+remoteAddr+":"+port+"\nPassword: "+password})

			// Garbage collect.
			//_ = cli.ContainerRemove(context.TODO(), res.ID, types.ContainerRemoveOptions{Force: true})
			//_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Database deleted.")

			// Return no errors.
			return nil
		},
	})
}
