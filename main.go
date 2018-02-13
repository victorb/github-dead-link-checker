package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/davecgh/go-spew/spew"
	mdParser "github.com/nikitavoloboev/markdown-parser/parser"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func urlIsOK(URL string) (bool, string, error) {
	response, err := http.Head(URL)
	if err != nil {
		return false, "Fatal Error", err
	}
	// Method not allowed
	if response.StatusCode == 405 {
		// try with get but with timeout both
		// before and after first received byte?
	}
	if response.StatusCode == 200 {
		return true, "", nil
	}
	return false, response.Status, nil
}

func main() {
	token := os.Getenv("GH_SECRET")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// list all repositories for the authenticated user
	repos, _, err := client.Repositories.ListByOrg(ctx, "ipfs", nil)
	if err != nil {
		panic(err)
	}
	for _, repo := range repos {
		fullName := repo.GetFullName()
		fmt.Println("## https://github.com/" + fullName)
		file, _, err := client.Repositories.GetReadme(ctx, "ipfs", repo.GetName(), nil)
		if err != nil {
			panic(err)
		}
		fileContent, err := file.GetContent()
		if err != nil {
			panic(err)
		}
		links := mdParser.GetAllLinks(fileContent)
		for _, link := range links {
			// if link is not starting with http, it's a relative link
			// / is absolute link
			// img/ipfs-alpha-video.png
			// github.com/ipfs/ipfs/blob/master/img/ipfs-alpha-video.png
			l, err := url.Parse(link)
			if l.Scheme == "" && l.Hostname() != "" {
				link = "https:" + link
			}
			// TODO need to handle
			if l.Hostname() == "" {
				link = "https://github.com/" + fullName + "/blob/master/" + link
			}
			isOK, errMsg, err := urlIsOK(link)
			if err != nil {
				errMsg = err.Error()
				spew.Dump(err)
				isOK = false
			}
			text := "FAIL"
			if isOK {
				text = "OK"
			}
			text = "  " + text + "  " + link
			if !isOK {
				text = text + " - " + errMsg
			}
			fmt.Println(text)
		}
	}
}
