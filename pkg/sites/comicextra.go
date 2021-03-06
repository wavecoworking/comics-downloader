package sites

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Girbons/comics-downloader/pkg/config"
	"github.com/Girbons/comics-downloader/pkg/core"
	"github.com/Girbons/comics-downloader/pkg/util"
	"github.com/anaskhan96/soup"
)

type Comicextra struct {
	options *config.Options
}

func NewComicextra(options *config.Options) *Comicextra {
	return &Comicextra{
		options: options,
	}
}

func (c *Comicextra) retrieveImageLinks(comic *core.Comic) ([]string, error) {
	var links []string

	response, err := soup.Get(comic.URLSource)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(util.IMAGEREGEX)
	match := re.FindAllStringSubmatch(response, -1)

	for i := range match {
		url := match[i][1]
		if util.IsURLValid(url) {
			links = append(links, url)
		}
	}

	if c.options.Debug {
		c.options.Logger.Debug(fmt.Sprintf("Image Links found: %s", strings.Join(links, " ")))
	}

	return links, err
}

func (c *Comicextra) isSingleIssue(url string) bool {
	return util.TrimAndSplitURL(url)[3] != "comic"
}

func (c *Comicextra) retrieveLastIssue(url string) (string, error) {
	var lastIssue string

	response, err := soup.Get(url)

	if err != nil {
		return "", err
	}

	doc := soup.HTMLParse(response)

	issues := doc.FindAll("option")
	if len(issues) != 0 {
		lastIssue = issues[len(issues)-1].Attrs()["value"]

		return lastIssue, nil
	}

	issues = doc.Find("tbody", "id", "list").FindAll("a")
	lastIssue = issues[0].Attrs()["href"]

	return lastIssue, nil
}

// RetrieveIssueLinks gets a slice of urls for all issues in a comic
func (c *Comicextra) RetrieveIssueLinks() ([]string, error) {
	url := c.options.Url

	if c.options.Last {
		issue, err := c.retrieveLastIssue(url)
		return []string{issue}, err
	}

	if c.options.All && c.isSingleIssue(url) {
		url = "https://www.comicextra.com/comic/" + util.TrimAndSplitURL(url)[3]
	} else if c.isSingleIssue(url) {
		return []string{url}, nil
	}

	// retrieve pages before

	name := util.TrimAndSplitURL(url)[4]
	var (
		pages    []string
		links    []string
		elements []soup.Root
	)

	// do not handle pagination
	// remove the page that comes within the url
	parts := strings.Split(url, "/")
	if len(parts) >= 6 {
		url = parts[0] + "//" + parts[2] + "/" + parts[3] + "/" + parts[4]
	}

	// and start from 1
	pages = append(pages, url+"/1")

	response, err := soup.Get(url)
	if err != nil {
		return nil, err
	}

	doc := soup.HTMLParse(response)

	pagesDiv := doc.Find("div", "class", "general-nav")
	if pagesDiv.Pointer != nil {
		elements = pagesDiv.FindAll("a")
	}

	for _, element := range elements {
		pageURL := element.Attrs()["href"]
		if !strings.Contains(pageURL, "onclick") && !util.IsValueInSlice(pageURL, pages) {
			pages = append(pages, pageURL)
		}
	}

	re := regexp.MustCompile("<a[^>]+href=\"([^\">]+" + "/" + name + "/.+)\"")

	for _, pageURL := range pages {
		response, err := soup.Get(pageURL)
		if err != nil {
			return nil, err
		}

		match := re.FindAllStringSubmatch(response, -1)

		for i := range match {
			url := match[i][1]
			if !strings.Contains(url, "onclick") && !util.IsValueInSlice(url, pages) {
				url = url + "/full"
				if util.IsURLValid(url) && !util.IsValueInSlice(url, links) {
					links = append(links, url)
				}
			}
		}
	}

	if c.options.Debug {
		c.options.Logger.Debug(fmt.Sprintf("Issues Links retrieved: %s", strings.Join(links, " ")))
	}

	return links, err
}

func (c *Comicextra) GetInfo(url string) (string, string) {
	parts := util.TrimAndSplitURL(url)
	name := parts[3]
	issueNumber := parts[4]

	return name, issueNumber
}

// Initialize will initialize the comic based
// on comicextra.com
func (c *Comicextra) Initialize(comic *core.Comic) error {
	links, err := c.retrieveImageLinks(comic)
	comic.Links = links

	return err
}
