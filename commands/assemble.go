package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"bytes"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
	"github.com/google/uuid"
	"github.com/jakemakesstuff/structuredhttp"
	"io/ioutil"
	"os"
	"os/exec"
)

func init() {
	router.Router.SetCommand(&gommand.Command{
		Name:        "assemble",
		Description: "Assembles a file in JASPer.",
		Usage:       "<attached assembler file>",
		Category:    categories.Informational,
		Function: func(ctx *gommand.Context) error {
			// Handle no attachments.
			if len(ctx.Message.Attachments) == 0 {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "You must attach a file to use this command.")
				return nil
			}

			// Handle the attachment.
			attachedFile := ctx.Message.Attachments[0]
			if attachedFile.Size > 1e+6 {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Your file is too large. Are you sure this is assembler?")
				return nil
			}

			// Download the attachment.
			resp, err := structuredhttp.GET(attachedFile.URL).Run()
			if err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Failed to get content:", err.Error())
				return nil
			}
			if err = resp.RaiseForStatus(); err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Failed to get content:", err.Error())
				return nil
			}
			b, err := resp.Bytes()
			if err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Failed to get content:", err.Error())
				return nil
			}

			// Save the content from the user.
			fileId := uuid.New().String()
			asmPath := "/root/" + fileId + ".s"
			jasPath := "/root/" + fileId + ".jas"
			err = ioutil.WriteFile(asmPath, b, 0777)
			if err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Failed to write file:", err.Error())
				return nil
			}
			defer func() {
				// Kill the files.
				_ = os.Remove(asmPath)
				_ = os.Remove(jasPath)
			}()

			// Call jasm.
			cmd := exec.Command("/bin/sh", "-c", "cd /root/jasper && perl ./jasm.pl -a "+asmPath+" -o "+jasPath)
			stderr := bytes.Buffer{}
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Command error:", err.Error())
				return nil
			}
			b, err = ioutil.ReadFile(jasPath)
			if err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Failed to read file:", err.Error())
				return nil
			}
			wasSuccessful := stderr.String() == "# Success\n"
			if wasSuccessful {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "👍 Successfully assembled!", &disgord.CreateMessageFileParams{
					Reader:   bytes.NewReader(b),
					FileName: "application.jas",
				})
			} else {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), "⚠️ Failed to assemble!", &disgord.CreateMessageFileParams{
					Reader:   bytes.NewReader(b),
					FileName: "errors.txt",
				})
			}

			// Return no errors.
			return nil
		},
	})
}
