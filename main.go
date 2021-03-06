package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/mitchellh/go-homedir"
)

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		commandPull,
		commandPush,
		commandPost,
	}

	app.Run(os.Args)
}

func loadConfigFile() *Config {
	home, err := homedir.Dir()
	dieIf(err)

	f, err := os.Open(filepath.Join(home, ".config", "blogsync", "config.yaml"))
	dieIf(err)

	conf, err := LoadConfig(f)
	dieIf(err)

	return conf
}

var commandPull = cli.Command{
	Name:  "pull",
	Usage: "Pull entries from remote",
	Action: func(c *cli.Context) {
		blog := c.Args().First()
		if blog == "" {
			cli.ShowCommandHelp(c, "pull")
			os.Exit(1)
		}

		conf := loadConfigFile()
		blogConfig := conf.Get(blog)
		if blogConfig == nil {
			logf("error", "blog not found: %s", blog)
			os.Exit(1)
		}

		b := NewBroker(blogConfig)
		remoteEntries, err := b.FetchRemoteEntries()
		dieIf(err)

		for _, re := range remoteEntries {
			path := b.LocalPath(re)
			_, err := b.StoreFresh(re, path)
			dieIf(err)
		}
	},
}

var commandPush = cli.Command{
	Name:  "push",
	Usage: "Push local entries to remote",
	Action: func(c *cli.Context) {
		path := c.Args().First()
		if path == "" {
			cli.ShowCommandHelp(c, "push")
			os.Exit(1)
		}

		path, err := filepath.Abs(path)
		dieIf(err)

		var blogConfig *BlogConfig

		conf := loadConfigFile()
		for remoteRoot := range conf.Blogs {
			bc := conf.Get(remoteRoot)
			localRoot, err := filepath.Abs(filepath.Join(bc.LocalRoot, remoteRoot))
			dieIf(err)

			if strings.HasPrefix(path, localRoot) {
				blogConfig = bc
				break
			}
		}

		if blogConfig == nil {
			logf("error", "cannot find blog for %s", path)
			os.Exit(1)
		}

		b := NewBroker(blogConfig)

		f, err := os.Open(path)
		dieIf(err)

		entry, err := entryFromReader(f)
		dieIf(err)

		b.UploadFresh(entry)
	},
}

var commandPost = cli.Command{
	Name:  "post",
	Usage: "Post a new entry to remote",
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "draft"},
		cli.StringFlag{Name: "title"},
		cli.StringFlag{Name: "custom-path"},
	},
	Action: func(c *cli.Context) {
		blog := c.Args().First()
		if blog == "" {
			cli.ShowCommandHelp(c, "post")
			os.Exit(1)
		}

		conf := loadConfigFile()
		blogConfig := conf.Get(blog)
		if blogConfig == nil {
			logf("error", "blog not found: %s", blog)
			os.Exit(1)
		}

		entry, err := entryFromReader(os.Stdin)
		dieIf(err)

		if c.Bool("draft") {
			entry.IsDraft = true
		}

		if path := c.String("custom-path"); path != "" {
			entry.CustomPath = path
		}

		if title := c.String("title"); title != "" {
			entry.Title = title
		}

		b := NewBroker(blogConfig)
		err = b.PostEntry(entry)
		dieIf(err)
	},
}
