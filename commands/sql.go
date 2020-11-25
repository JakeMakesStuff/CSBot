package commands

import (
	"CSBot/categories"
	"CSBot/router"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/auttaja/gommand"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/godror/godror"
	"github.com/hako/durafmt"
	"github.com/jakemakesstuff/structuredhttp"
	"github.com/olekukonko/tablewriter"
	"strconv"
	"strings"
	"time"
)

func formatQuery(res *sql.Rows) (formatted string, rows int) {
	// Close the rows when we're done.
	defer res.Close()

	// Return nothing.
	setNothing := func() {
		rows = 0
		formatted = "<no data was returned>"
	}

	// Get the columns.
	columns, err := res.Columns()
	if err != nil || len(columns) == 0 {
		setNothing()
		return
	}
	x := &bytes.Buffer{}
	writer := tablewriter.NewWriter(x)
	writer.SetHeader(columns)

	for res.Next() {
		// Defines the initial row.
		row := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(row))
		for i := range row {
			ptrs[i] = &row[i]
		}

		// Run the scan.
		err := res.Scan(ptrs...)
		if err != nil {
			formatted = "decoding error: " + err.Error()
		}

		// Get the row as string.
		stringRow := make([]string, len(row))
		for i, v := range row {
			stringRow[i] = fmt.Sprint(v)
		}

		// Add to the writer.
		writer.Append(stringRow)

		// Add 1 to rows.
		rows++
	}

	// Flush the writer.
	writer.Render()

	// Set it to formatted.
	formatted = x.String()

	// Return everything.
	return
}

func sqlAwareSplit(text string) []string {
	// Get the indexes.
	indexes := make([]int, 0)
	inQuote := false
	escaped := false
	for i, v := range text {
		if v == '\\' {
			// Handle escapes.
			escaped = !escaped
		} else if v == '\'' {
			// Handle a quote.
			if escaped {
				escaped = false
			} else {
				inQuote = !inQuote
			}
		} else if v == ';' {
			// Handle a semi-colon.
			if escaped {
				escaped = false
			} else {
				indexes = append(indexes, i)
			}
		} else {
			// Handle anything else.
			if escaped {
				escaped = false
			}
		}
	}

	// If len of indexes is 0, just return text.
	if len(indexes) == 0 {
		return []string{text}
	}

	// If not, split at each index.
	beginning := -1
	parts := make([]string, 0)
	for _, v := range indexes {
		parts = append(parts, text[beginning+1:v])
		beginning = v
	}
	return parts
}

func processMd(text string) string {
	blockStart := 0
	blockTotal := 0
	start := true
	for i, v := range text {
		if v == '`' {
			if start {
				// Handle processing the blocks.
				if blockStart == 3 {
					// Set start to false.
					start = false
				} else {
					// Add to the blocks.
					blockStart++
					blockTotal++
				}
			} else {
				// Subtract 1 from block total.
				blockTotal--
				if blockTotal == 0 {
					// This is our block. Process this.
					text = text[blockStart : len(text)-blockStart]
					if strings.HasPrefix(text, "sql\n") {
						text = text[4:]
					}
					return text
				}
			}
		} else {
			if i == 0 {
				// Not code block.
				return text
			}
			start = false
			if blockTotal != blockStart {
				// Reset blocks.
				blockTotal = blockStart
			}
		}
	}
	return text
}

func init() {
	cli, err := client.NewClient("unix:///var/run/docker.sock", "", nil, nil)
	if err != nil {
		fmt.Println("Unable to connect to Docker:", err.Error())
	}

	router.Router.SetCommand(&gommand.Command{
		Name:        "sql",
		Description: "Start a sandboxed Oracle SQL environment.",
		Usage:       "[use mysql (true/false, defaults to false)]",
		Category:    categories.Learning,
		ArgTransformers: []gommand.ArgTransformer{
			{
				Function: gommand.BooleanTransformer,
				Optional: true,
			},
		},
		Function: func(ctx *gommand.Context) error {
			// Defines if we should use mysql.
			mysql, _ := ctx.Args[0].(bool)

			// Handle if not configured.
			if cli == nil {
				_, _ = ctx.Reply("Docker is not configured.")
				return nil
			}

			// Send the initial embed for starting the container.
			msg, err := ctx.Reply(&disgord.Embed{
				Title:       "Creating Docker Container...",
				Description: "Creating a Docker container which contains your database.",
			})
			if err != nil {
				return nil
			}
			image := "store/oracle/database-enterprise:12.2.0.1-slim"
			var env []string
			if mysql {
				image = "healthcheck/mysql"
				env = []string{"MYSQL_ALLOW_EMPTY_PASSWORD=yes", "MYSQL_DATABASE=sandbox"}
			}
			res, err := cli.ContainerCreate(context.TODO(), &container.Config{
				Image: image,
				Env:   env,
			}, nil, nil, msg.ID.String())
			if err != nil {
				_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
					Title:       "Failed to launch container.",
					Description: err.Error(),
				}).Execute()
				return nil
			}
			err = cli.ContainerStart(context.TODO(), res.ID, types.ContainerStartOptions{})
			if err != nil {
				_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
					Title:       "Failed to start container.",
					Description: err.Error(),
				}).Execute()
				return nil
			}

			// Send an embed whilst we wait for the DB to initialise.
			_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
				Title:       "Waiting for the database to initialise...",
				Description: "Waiting for the database container to report as healthy.",
			}).Execute()
			var info types.ContainerJSON
			for {
				// Get the container info.
				info, err = cli.ContainerInspect(context.TODO(), res.ID)
				if err != nil {
					_ = cli.ContainerRemove(context.TODO(), res.ID, types.ContainerRemoveOptions{Force: true})
					_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
						Title:       "Failed to get container info.",
						Description: err.Error(),
					}).Execute()
					return nil
				}

				// Check if the container has stopped.
				if !info.State.Running {
					_ = cli.ContainerRemove(context.TODO(), res.ID, types.ContainerRemoveOptions{Force: true})
					_, _ = ctx.Session.UpdateMessage(context.TODO(), msg.ChannelID, msg.ID).SetEmbed(&disgord.Embed{
						Title:       "The container died.",
						Description: "The container seems to have died during birth. Please try rerunning this command.",
					}).Execute()
					return nil
				}

				// Check if the container is healthy.
				if info.State.Health.Status == "healthy" {
					break
				}

				// Sleep for 100ms.
				time.Sleep(100 * time.Millisecond)
			}

			// Attempt to connect to the DB.
			dataSourceName := `user="system" password="Oradoc_db1" connectString="` + info.NetworkSettings.IPAddress + `/ORCLCDB.localdomain"`
			driverName := "godror"
			if mysql {
				driverName = "mysql"
				dataSourceName = "root:@tcp(" + info.NetworkSettings.IPAddress + ")/sandbox"
			}
			db, err := sql.Open(driverName, dataSourceName)
			if err != nil {
				_, _ = ctx.Reply(ctx.Message.Author.Mention(), err.Error())
				return nil
			}

			// Loop until a ping succeeds to get around a bug with the Oracle DB official docker image where the user isn't immediately made when the node reports healthy.
			if !mysql {
				for {
					if err := db.Ping(); err == nil {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
			}

			// Delete the old message.
			_ = ctx.Session.DeleteMessage(context.TODO(), msg.ChannelID, msg.ID)

			// Used to handle processing a query.
			processQuery := func(content string) (formatted string, timeTaken *time.Duration, rows int, err error) {
				var stmt *sql.Stmt
				stmt, err = db.Prepare(content)
				t1 := time.Now()
				var res *sql.Rows
				if err == nil {
					res, err = stmt.Query()
				}
				t2 := time.Now()
				x := t2.Sub(t1)
				timeTaken = &x
				if err == nil {
					formatted, rows = formatQuery(res)
				}
				return
			}

			// Launch eval mode.
			_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Database created. You are now in eval mode. To perform database queries, simply type or attach your SQL query. To exit this mode, simply type `exit`. Note you will automatically be taken out of eval mode and the database removed after 10 minutes of user inactivity.")
			for {
				// Try and get the message.
				c, _ := context.WithTimeout(context.TODO(), time.Minute*10)
				m := ctx.WaitManager.WaitForMessageCreate(c, func(s disgord.Session, evt *disgord.MessageCreate) bool {
					if evt.Message.Author.ID == ctx.Message.Author.ID {
						return true
					}
					return false
				})
				if m == nil {
					break
				}

				// If the message content is exit, break here.
				if strings.ToLower(m.Message.Content) == "exit" {
					break
				}

				// Check if there's content.
				text := ""
				if m.Message.Content == "" {
					// We should try and grab the attachment.
					if len(m.Message.Attachments) == 0 {
						// Ok, go back to the top.
						continue
					}
					if m.Message.Attachments[0].Size > 1000000 {
						// In this situation, we should assume abuse.
						continue
					}
					resp, err := structuredhttp.GET(m.Message.Attachments[0].URL).Run()
					if err != nil {
						continue
					}
					text, _ = resp.Text()
				} else {
					// We will use this as the text.
					text = processMd(m.Message.Content)
				}
				if text != "" {
					// Handle the specified text.
					commaSplit := sqlAwareSplit(text)
					queryTimes := make([]time.Duration, 0, len(commaSplit))
					formattedResults := make([]string, 0, len(commaSplit))
					totalRows := 0
					createFooter := func() *disgord.EmbedFooter {
						var timesAdded time.Duration
						for _, v := range queryTimes {
							timesAdded += v
						}
						return &disgord.EmbedFooter{
							Text: strconv.Itoa(totalRows) + " rows affected | Time taken: " + durafmt.Parse(timesAdded).String() + " | use `exit` to exit eval mode",
						}
					}

					// Go through each item and run it.
					outerContinue := false
					for _, v := range commaSplit {
						v = strings.TrimSpace(v)
						if v != "" {
							// Get the result.
							formatted, timeTaken, rows, err := processQuery(v)

							// Add the time taken to the query times if it isn't nil.
							if timeTaken != nil {
								queryTimes = append(queryTimes, *timeTaken)
							}

							// Add rows to the total rows.
							totalRows += rows

							// If formatted isn't blank, add this to the results.
							if formatted != "" {
								formattedResults = append(formattedResults, formatted)
							}

							// Handle error.
							if err != nil {
								// The additional information if there's queries which succeeded.
								additionalInfo := ""
								if len(formattedResults) != 0 {
									additionalInfo = " Note that some SQL queries before this one did succeed, and will be attached."
								}

								// Define the arguments we will parse to disgord.
								x := []interface{}{
									ctx.Message.Author.Mention(),
									&disgord.Embed{
										Title:       "SQL query failed",
										Description: "```" + err.Error() + "```\n" + additionalInfo,
										Footer:      createFooter(),
										Color:       0xFF0000,
									},
								}
								if len(formattedResults) != 0 {
									// Defines the content.
									content := ""

									// Render the file.
									for i, v := range formattedResults {
										content += "---\nQuery " + strconv.Itoa(i) + " (took " + durafmt.Parse(queryTimes[i]).String() + "):\n---\n" + v + "\n"
									}

									// Add the file.
									x = append(x, &disgord.CreateMessageFileParams{
										Reader:   strings.NewReader(content),
										FileName: "before_queries.txt",
									})
								}

								// Send the message and break.
								outerContinue = true
								_, _ = ctx.Reply(x...)
								break
							}
						}
					}
					if outerContinue {
						continue
					}

					// Render the queries.
					content := ""
					if len(formattedResults) == 1 {
						// Just set content to the 1 result.
						content = formattedResults[0]
					} else {
						// Format the results.
						for i, v := range formattedResults {
							content += "---\nQuery " + strconv.Itoa(i) + " (took " + durafmt.Parse(queryTimes[i]).String() + "):\n---\n" + v + "\n"
						}
					}
					description := "```\n" + content + "```"
					large := false
					if len(description) > 2048 {
						large = true
						description = "<too large for Discord embed, see attached file>"
					}
					// Define the arguments we will parse to disgord.
					x := []interface{}{
						ctx.Message.Author.Mention(),
						&disgord.Embed{
							Title:       "SQL query result",
							Description: description,
							Footer:      createFooter(),
							Color:       0x00FF00,
						},
					}
					if large {
						x = append(x, &disgord.CreateMessageFileParams{
							Reader:     strings.NewReader(content),
							FileName:   "content.txt",
							SpoilerTag: false,
						})
					}
					_, _ = ctx.Reply(x...)
				}
			}

			// Garbage collect.
			_ = cli.ContainerRemove(context.TODO(), res.ID, types.ContainerRemoveOptions{Force: true})
			_, _ = ctx.Reply(ctx.Message.Author.Mention(), "Database deleted.")

			// Return no errors.
			return nil
		},
	})
}
