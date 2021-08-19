package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/hashicorp/go-changelog"
)

func main() {
	ctx := context.Background()

	var outputPath string
	var pullRequestNumber int
	flag.StringVar(&outputPath, "output-path", "", "if not empty, directory to which extracted notes file should be written")
	flag.IntVar(&pullRequestNumber, "pull-request", 0, "number of pull request to be read")
	flag.Parse()

	if pullRequestNumber == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Must specify pull request number.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	repository := os.Getenv("GITHUB_REPOSITORY")
	if repository == "" {
		log.Fatalf("GITHUB_REPOSITORY not set")
	}
	repositorySplit := strings.Split(repository, "/")
	if len(repositorySplit) != 2 || repositorySplit[0] == "" || repositorySplit[1] == "" {
		log.Fatalf("GITHUB_REPOSITORY should be of the form <owner>/<repo>")
	}
	owner := repositorySplit[0]
	repo := repositorySplit[1]

	httpClient := http.DefaultClient
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		token := oauth2.Token{AccessToken: token}
		ts := oauth2.StaticTokenSource(&token)
		httpClient = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(httpClient)

	pullRequest, _, err := client.PullRequests.Get(ctx, owner, repo, pullRequestNumber)
	if err != nil {
		log.Fatalf("Error retrieving pull request github.com/"+
			"%s/%s/%d: %s", owner, repo, pullRequestNumber, err)
	}
	entry := changelog.Entry{
		Issue: strconv.Itoa(pullRequestNumber),
		Body:  pullRequest.GetBody(),
	}
	notes := changelog.NotesFromEntry(entry)
	if len(notes) < 1 {
		log.Printf("no changelog entry found in %s: %s", entry.Issue,
			string(entry.Body))
		body := "It looks like no changelog entry is attached to" +
			" this PR. Please include a release note block" +
			" in the PR body, as described in https://github.com/GoogleCloudPlatform/magic-modules/blob/master/.ci/RELEASE_NOTES_GUIDE.md:" +
			"\n\n~~~\n```release-note:TYPE\nRelease note" +
			"\n```\n~~~"
		_, _, err = client.Issues.CreateComment(ctx, owner, repo,
			pullRequestNumber, &github.IssueComment{
				Body: &body,
			})
		if err != nil {
			log.Fatalf("Error creating pull request comment on"+
				" github.com/%s/%s/%d: %s", owner, repo, pullRequestNumber,
				err)
		}
		os.Exit(1)
	}

	var unknownTypes []string
	for _, note := range notes {
		switch note.Type {
		case "none",
			"warning",
			"feature",
			"bug",
			"other",
			"documentation",
			"testing",
			"unknown":
		default:
			unknownTypes = append(unknownTypes, note.Type)
		}
	}
	if len(unknownTypes) > 0 {
		log.Printf("unknown changelog types %v", unknownTypes)
		body := "It looks like you're using"
		if len(unknownTypes) == 1 {
			body += " an"
		}
		body += " unknown release-note type"
		if len(unknownTypes) > 1 {
			body += "s"
		}
		body += " in your changelog entries:"
		for _, t := range unknownTypes {
			body += "\n* " + t
		}
		body += "\n\nPlease only use the types listed in https://github.com/GoogleCloudPlatform/magic-modules/blob/master/.ci/RELEASE_NOTES_GUIDE.md."
		_, _, err = client.Issues.CreateComment(ctx, owner, repo,
			pullRequestNumber, &github.IssueComment{
				Body: &body,
			})
		if err != nil {
			log.Fatalf("Error creating pull request comment on"+
				" github.com/%s/%s/%d: %s", owner, repo, pullRequestNumber,
				err)
		}
		os.Exit(1)
	}

	if outputPath == "" {
		return
	}

	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		log.Fatalf("Failed to ensure directory exists: %s", err)
	}

	fileContent := changelog.NotesToString(notes)
	filename := fmt.Sprintf("%d.txt", pullRequestNumber)
	outputFile := filepath.Join(outputPath, filename)

	if fileContent != "" {
		err = os.WriteFile(outputFile, []byte(fileContent), 0644)
		if err != nil {
			log.Fatalf("Failed to write file: %s", err)
		}
	}
}
