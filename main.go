package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	mdParser "github.com/nikitavoloboev/markdown-parser/parser"
)

type Repository struct {
	Organization string
	Name         string
}

type CheckLink struct {
	Repository Repository
	Link       string
	Result     string
}

func (j *Repository) GetFullName() string {
	return j.Organization + "/" + j.Name
}

func (j *Repository) SetFullName(name string) *Repository {
	split := strings.Split(name, "/")
	j.Organization = split[0]
	j.Name = split[1]
	return j
}

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
	runtime.GOMAXPROCS(runtime.NumCPU())
	var workers = flag.Int("workers", 10, "Number of workers to concurrently check links")
	flag.Parse()

	args := flag.Args()
	reposToTest := []Repository{}
	wantedOrgs := []string{}
	wantedRepos := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "/") {
			fmt.Printf("Using `%s` as a repository\n", arg)
			wantedRepos = append(wantedRepos, arg)
		} else {
			fmt.Printf("Using `%s` as a organization\n", arg)
			wantedOrgs = append(wantedOrgs, arg)
		}
	}
	fmt.Println()
	for _, repo := range wantedRepos {
		split := strings.Split(repo, "/")
		reposToTest = append(reposToTest, Repository{
			Organization: split[0],
			Name:         split[1],
		})
	}
	token := os.Getenv("GH_SECRET")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// list all repositories for the authenticated user
	for _, org := range wantedOrgs {
		repos, _, err := client.Repositories.ListByOrg(ctx, org, nil)
		if err != nil {
			panic(err)
		}
		for _, repo := range repos {
			reposToTest = append(reposToTest, Repository{
				Organization: org,
				Name:         repo.GetName(),
			})
		}
	}
	errors := []string{}
	errorCh := make(chan string)
	go func() {
		for {
			err := <-errorCh
			errors = append(errors, err)
		}
	}()
	linksToCheck := make(chan CheckLink)
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		go func() {
			for {
				link := <-linksToCheck
				isOK, errMsg, err := urlIsOK(link.Link)
				if err != nil {
					errMsg = err.Error()
					spew.Dump(err)
					isOK = false
				}
				text := red("  FAIL ")
				if isOK {
					text = green("  OK   ")
				}
				fullName := link.Repository.GetFullName()
				text = text + fullName + " " + link.Link
				if !isOK {
					text = text + " - " + errMsg
					errorCh <- text
				}
				fmt.Println(text)
				wg.Done()
			}
		}()
	}
	for _, repo := range reposToTest {
		fullName := repo.GetFullName()
		file, _, err := client.Repositories.GetReadme(ctx, repo.Organization, repo.Name, nil)
		if err != nil {
			if strings.Contains(err.Error(), "404 Not Found") {
				continue
			} else {
				panic(err)
			}
		}
		fileContent, err := file.GetContent()
		if err != nil {
			panic(err)
		}
		links := mdParser.GetAllLinks(fileContent)
		parsedLinks := []string{}
		for _, link := range links {
			l, err := url.Parse(link)
			if err != nil {
				panic(err)
			}
			if l.Scheme == "" && l.Hostname() != "" {
				link = "https:" + link
			}
			// TODO need to handle
			if l.Hostname() == "" {
				link = "https://github.com/" + fullName + "/blob/master/" + link
			}
			parsedLinks = append(parsedLinks, link)
		}
		wg.Add(len(parsedLinks))
		for _, link := range parsedLinks {
			cl := CheckLink{}
			cl.Repository.SetFullName(fullName)
			cl.Link = link
			go func() {
				linksToCheck <- cl
			}()
		}
	}
	wg.Wait()
	if len(errors) > 0 {
		fmt.Println()
		fmt.Println("ALL ERRORS:")
		for _, err := range errors {
			fmt.Println(err)
		}
	} else {
		fmt.Println("All good, found no broken links")
	}
}
