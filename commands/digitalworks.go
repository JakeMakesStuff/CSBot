package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"context"
	"errors"
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
	"strings"
	"time"
)

func init() {
	rand.NewSource(time.Now().UnixNano())

	cli, err := client.NewClient("unix:///var/run/docker.sock", "", nil, nil)
	if err != nil {
		fmt.Println("Unable to connect to Docker:", err.Error())
	}

	router.Router.SetCommand(&gommand.Command{
		Name:                 "digitalworks",
		Description:          "Start a sandboxed DigitalWorks VNC environment. You will then get DM'd the VNC hostname and password.",
		Usage: "[width (defaults to 800)] [height (defaults to 600)]",
		Category:             categories.Learning,
		ArgTransformers: []gommand.ArgTransformer{
			{
				Function: gommand.UIntTransformer,
				Optional: true,
			},
			{
				Function: gommand.UIntTransformer,
				Optional: true,
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

			// Check if the container already exists. If so manage showing the user options relating to this.
			containerName := ctx.Message.Author.ID.String() + "-vnc"
			c, err := cli.ContainerInspect(context.TODO(), containerName)
			if err == nil {
				message := "You already have a DigitalWorks container running. This means you have 2 options:\n\n" +
					"♻️ **Destroy the container:** You will want to do this if you want to change the resolution of your container. Note that this will not destroy your persistent folder on your desktop, but will destroy all other container content.\n" +
					"✉️ **Re-send the credentials:** Resends the login credentials in a DM."
				msg, err := ctx.Reply(message)
				if err == nil {
					deadline, _ := context.WithTimeout(context.TODO(), time.Minute*10)
					r := ctx.WaitManager.WaitForMessageReactionAdd(deadline, func(_ disgord.Session, evt *disgord.MessageReactionAdd) bool {
						println(evt.PartialEmoji.String())
						fmt.Println(evt)
						fmt.Println(msg)
						return evt.MessageID == msg.ID && evt.UserID == ctx.Message.Author.ID && (evt.PartialEmoji.String() == "♻" || evt.PartialEmoji.String() == "✉️")
					})
					_ = ctx.Session.DeleteMessage(context.TODO(), msg.ChannelID, msg.ID)
					if r != nil {
						if r.PartialEmoji.String() == "♻" {
							// Destroy the container.
							_ = cli.ContainerRemove(context.TODO(), c.ID, types.ContainerRemoveOptions{Force: true})
							_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Database deleted.")
						} else {
							// Resend the credentials via DMs.
							conn, err := net.Dial("udp", "8.8.8.8:80")
							if err != nil {
								log.Fatal(err)
							}
							defer conn.Close()
							remoteAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
							var password string
							for _, v := range c.Config.Env {
								if strings.HasPrefix(v, "VNC_PASSWORD=") {
									password = v[13:]
									break
								}
							}
							if password == "" {
								return errors.New("password field is blank for some weird reason")
							}
							port := c.HostConfig.PortBindings["5900/tcp"][0].HostPort
							_, _, _ = ctx.Message.Author.SendMsg(context.TODO(), ctx.Session, &disgord.Message{Content: "Hostname: "+remoteAddr+":"+port+"\nPassword: "+password})
						}
					}
				}
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
			}, &container.HostConfig{PortBindings: portMap}, nil, containerName)
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
			remoteAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
			_, _, _ = ctx.Message.Author.SendMsg(context.TODO(), ctx.Session, &disgord.Message{Content: "Hostname: "+remoteAddr+":"+port+"\nPassword: "+password})

			// Return no errors.
			return nil
		},
	})
}
