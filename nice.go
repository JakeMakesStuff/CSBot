package main

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/jakemakesstuff/structuredhttp"
	"github.com/otiai10/gosseract"
	"strings"
)

func messageCreateNice(s disgord.Session, evt *disgord.MessageCreate) {
	if evt.Message.Author.Bot || len(evt.Message.Attachments) == 0 {
		return
	}
	go func() {
		// Check for any images.
		images := make([]*disgord.Attachment, 0, len(evt.Message.Attachments))
		exts := []string{"png", "jpg", "jpeg"}
		for _, v := range evt.Message.Attachments {
			filenameLower := strings.ToLower(v.Filename)
			for _, ext := range exts {
				if strings.HasSuffix(filenameLower, ext) {
					images = append(images, v)
					break
				}
			}
		}
		if len(images) == 0 {
			// No images to process.
			return
		}

		// Process each possible image.
		for _, v := range images {
			go func(v *disgord.Attachment) {
				// Try and get the contents of the image.
				resp, err := structuredhttp.GET(v.URL).Run()
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				if err = resp.RaiseForStatus(); err != nil {
					fmt.Println(err.Error())
					return
				}
				b, err := resp.Bytes()
				if err != nil {
					fmt.Println(err.Error())
					return
				}

				// Load image into gosseract and get the text content.
				gClient := gosseract.NewClient()
				if err = gClient.SetImageFromBytes(b); err != nil {
					return
				}
				if text, err := gClient.Text(); err == nil {
					if strings.Contains(text, "69") {
						niceLog(text)
						_, _ = s.SendMsg(context.TODO(), evt.Message.ChannelID, evt.Message.Author.Mention(), "nice")
					}
				} else {
					fmt.Println(err.Error())
				}
			}(v)
		}
	}()
}
