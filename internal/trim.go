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

		n := notion.NewClient(c.String("token"))

		page, err := n.FindPageByID(context.Background(), c.Args().First())
		if err != nil {
			return cli.Exit(err, 2)
		}

		if c.Bool("recursive") {
			fmt.Fprintf(c.App.Writer, "trimming page titles recursively, any errors will not lead to non-zero exit code\n")

			for _, err := range trimPageTitlesRecursively(c, n, []notion.Page{page}) {
				if err != nil {
					fmt.Fprintf(c.App.ErrWriter, "error trimming page %s: %s\n", page.ID, err)
				}
			}
		} else {
			if err := trimPageTitle(c, n, page); err != nil {
				return cli.Exit(err, 2)
			}
		}

		return nil
	},
}

func trimPageTitlesRecursively(c *cli.Context, n *notion.Client, pages []notion.Page) []error {
	var errs []error
	var nextLevelPages []notion.Page

	for {
		if len(pages) < 1 {
			break
		}

		for _, page := range pages {
			if err := trimPageTitle(c, n, page); err != nil {
				errs = append(errs, err)
			}

			nextLevelQuery := &notion.PaginationQuery{}
			for {
				// not sure if this is the right way to fetch child pages, but Notion API is weird
				children, err := n.FindBlockChildrenByID(context.Background(), page.ID, nextLevelQuery)
				if err != nil {
					errs = append(errs, err)
					break
				}

				for _, child := range children.Results {
					if child.Type != notion.BlockTypeChildPage {
						continue
					}
					childPage, err := n.FindPageByID(context.Background(), child.ID)
					if err != nil {
						errs = append(errs, err)
					} else {
						nextLevelPages = append(nextLevelPages, childPage)
					}
				}

				if children.HasMore {
					nextLevelQuery.StartCursor = *children.NextCursor
					continue
				}

				break
			}
		}

		pages = nextLevelPages
		nextLevelPages = []notion.Page{}
	}

	return errs
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
		plainText.WriteString(part.PlainText)
		if i < len(richText)-1 {
			plainText.WriteRune('•')
		}
	}

	return plainText.String()
}
