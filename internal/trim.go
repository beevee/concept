package internal

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/dstotijn/go-notion"
	"github.com/urfave/cli/v2"
)

var TrimCommand = cli.Command{
	Name:      "trim",
	Usage:     "remove leading and trailing spaces from page title",
	ArgsUsage: "<page_id>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "recursive",
			Aliases: []string{"r"},
			Usage:   "process child pages recursively",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return cli.Exit("please provide page id", 1)
		}

		client := notion.NewClient(c.String("token"))

		page, err := client.FindPageByID(context.Background(), c.Args().First())
		if err != nil {
			return cli.Exit(err, 2)
		}

		if c.Bool("recursive") {
			fmt.Fprintf(c.App.Writer, "trimming page titles recursively, any errors will not lead to non-zero exit code\n")

			var errs []error
			errs = append(errs, trimPageTitle(c, client, page))

			// here I need to fetch child pages, but not sure if it's possible

			for _, err := range errs {
				if err != nil {
					fmt.Fprintf(c.App.ErrWriter, "error trimming page %s: %s\n", page.ID, err)
				}
			}
		} else {
			if err := trimPageTitle(c, client, page); err != nil {
				return cli.Exit(err, 2)
			}
		}

		return nil
	},
}

func trimPageTitle(c *cli.Context, client *notion.Client, page notion.Page) error {
	if page.Parent.Type == notion.ParentTypeDatabase {
		return fmt.Errorf("page type %s not supported", page.Parent.Type)
	}

	title := page.Properties.(notion.PageProperties).Title.Title
	fmt.Fprintf(c.App.Writer, "trimming title for page %s (%s)\n", page.ID, renderRichTextAsPlain(title))

	firstPartIndex := 0
	for {
		if title[firstPartIndex].Type != notion.RichTextTypeText {
			break
		}
		title[firstPartIndex].Text.Content = strings.TrimLeftFunc(title[firstPartIndex].Text.Content, unicode.IsSpace)
		if len(title[firstPartIndex].Text.Content) > 0 {
			break
		}
		firstPartIndex += 1
	}

	lastPartIndex := len(title) - 1
	for {
		if title[lastPartIndex].Type != notion.RichTextTypeText {
			break
		}
		title[lastPartIndex].Text.Content = strings.TrimRightFunc(title[lastPartIndex].Text.Content, unicode.IsSpace)
		if len(title[lastPartIndex].Text.Content) > 0 {
			break
		}
		lastPartIndex -= 1
	}

	title = title[firstPartIndex : lastPartIndex+1]

	_, err := client.UpdatePageProps(context.Background(), page.ID, notion.UpdatePageParams{Title: title})
	return err
}

func renderRichTextAsPlain(richText []notion.RichText) string {
	plainText := strings.Builder{}

	for i, part := range richText {
		plainText.WriteString(part.Text.Content)
		if i < len(richText)-1 {
			plainText.WriteRune('•')
		}
	}

	return plainText.String()
}
