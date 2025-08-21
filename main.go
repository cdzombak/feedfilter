package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	ec "github.com/cdzombak/exitcode_go"
	"github.com/google/cel-go/cel"
	"github.com/gopherlibs/feedhub/feedhub"
	"github.com/mmcdole/gofeed"
)

const (
	ParseTimeout = 20 * time.Second
	OutMode      = os.FileMode(0644)

	FmtRss  = "rss"
	FmtAtom = "atom"
	FmtJson = "json"
)

var version = "dev"

type Config struct {
	From      string `json:"from"`       // URL to read from
	To        string `json:"to"`         // File to write to
	ToFmt     string `json:"to_fmt"`     // Format to write to (json, rss, atom)
	IncludeIf string `json:"include_if"` // CEL expression evaluated to determine whether each item should be included. If empty, all items are included.
	Meta      struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Link        string `json:"link"`
	} `json:"meta"`
}

func main() {
	configPath := flag.String("config", "./config.json", "Path to config file")
	printVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *printVersion {
		fmt.Printf("feedfilter version %s\n", version)
		os.Exit(ec.Success)
	}

	cfgBytes, err := os.ReadFile(*configPath)
	if err != nil {
		log.Printf("Error reading config file: %s", err)
		os.Exit(ec.NotConfigured)
	}
	var cfg Config
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		log.Printf("Error parsing config file: %s", err)
		os.Exit(ec.NotConfigured)
	}

	if //goland:noinspection HttpUrlsUsage
	cfg.From == "" || (!strings.HasPrefix(cfg.From, "http://") && !strings.HasPrefix(cfg.From, "https://")) {
		log.Println("Config error: 'from' must be a valid URL")
		os.Exit(ec.NotConfigured)
	}
	if cfg.To == "" {
		log.Println("Config error: 'to' must be a valid file path or '-' for stdout")
		os.Exit(ec.NotConfigured)
	}
	if cfg.ToFmt != FmtJson && cfg.ToFmt != FmtRss && cfg.ToFmt != FmtAtom {
		log.Println("Config error: 'to_fmt' must be one of 'json', 'rss', or 'atom'")
		os.Exit(ec.NotConfigured)
	}
	if cfg.Meta.Title == "" {
		cfg.Meta.Title = "$$ORIG$$"
	}
	if cfg.Meta.Description == "" {
		cfg.Meta.Description = "$$ORIG$$"
	}

	celEnv, err := cel.NewEnv(
		cel.Variable("title", cel.StringType),
		cel.Variable("description", cel.StringType),
		cel.Variable("link", cel.StringType),
		cel.Variable("categories", cel.ListType(cel.StringType)),
	)
	if err != nil {
		log.Printf("Error creating CEL environment: %s", err)
		os.Exit(ec.Failure)
	}
	celAST, issues := celEnv.Compile(cfg.IncludeIf)
	if issues != nil && issues.Err() != nil {
		log.Printf("Config error: include_if type-check error: %s", issues.Err())
		os.Exit(ec.NotConfigured)
	}
	celPrg, err := celEnv.Program(celAST)
	if err != nil {
		log.Printf("CEL program construction error: %s", err)
		os.Exit(ec.Failure)
	}

	parseCtx, parseCtxCancel := context.WithTimeout(context.Background(), ParseTimeout)
	defer parseCtxCancel()
	parser := gofeed.NewParser()
	parser.UserAgent = "github.com/cdzombak/feedfilter|" + version
	inFeed, err := parser.ParseURLWithContext(cfg.From, parseCtx)
	if err != nil {
		log.Printf("Error parsing feed: %s", err)
		os.Exit(ec.Failure)
	}

	var includeItems []*gofeed.Item
	for _, item := range inFeed.Items {
		out, _, err := celPrg.Eval(map[string]interface{}{
			"title":       item.Title,
			"description": item.Description,
			"link":        item.Link,
			"categories":  item.Categories,
		})
		if err != nil {
			log.Printf("Error evaluating include_if expression: %s", err)
			os.Exit(ec.Failure)
		}
		if out != nil {
			outVal, err := out.ConvertToNative(reflect.TypeOf(true))
			if err != nil {
				log.Printf("Error converting include_if result to bool: %s", err)
				os.Exit(ec.Failure)
			}
			if include, ok := outVal.(bool); ok && include {
				includeItems = append(includeItems, item)
			}
		}
	}

	outFeed := &feedhub.Feed{
		Title:       strings.ReplaceAll(cfg.Meta.Title, "$$ORIG$$", inFeed.Title),
		Description: strings.ReplaceAll(cfg.Meta.Description, "$$ORIG$$", inFeed.Description),
		Copyright:   inFeed.Copyright,
	}
	if cfg.Meta.Link != "" {
		outFeed.Link = &feedhub.Link{Href: cfg.Meta.Link}
	} else if inFeed.Link != "" {
		outFeed.Link = &feedhub.Link{Href: inFeed.Link}
	}
	if len(inFeed.Authors) > 0 {
		outFeed.Author = &feedhub.Author{
			Name:  inFeed.Authors[0].Name,
			Email: inFeed.Authors[0].Email,
		}
	}
	if inFeed.UpdatedParsed != nil {
		outFeed.Updated = *inFeed.UpdatedParsed
	}
	if inFeed.PublishedParsed != nil {
		outFeed.Created = *inFeed.PublishedParsed
	}
	if inFeed.Image != nil {
		outFeed.Image = &feedhub.Image{
			Url:   inFeed.Image.URL,
			Title: inFeed.Image.Title,
		}
	}

	for _, inItem := range includeItems {
		outItem := &feedhub.Item{
			Title:       inItem.Title,
			Link:        &feedhub.Link{Href: inItem.Link},
			Description: inItem.Description,
			Id:          inItem.GUID,
			Content:     inItem.Content,
		}
		if len(inItem.Authors) > 0 {
			outItem.Author = &feedhub.Author{
				Name:  inItem.Authors[0].Name,
				Email: inItem.Authors[0].Email,
			}
		}
		if inItem.UpdatedParsed != nil {
			outItem.Updated = *inItem.UpdatedParsed
		}
		if inItem.PublishedParsed != nil {
			outItem.Created = *inItem.PublishedParsed
		}
		if len(inItem.Enclosures) > 0 {
			outItem.Enclosure = &feedhub.Enclosure{
				Url:    inItem.Enclosures[0].URL,
				Length: inItem.Enclosures[0].Length,
				Type:   inItem.Enclosures[0].Type,
			}
		}
		if len(inItem.Categories) > 0 {
			outItem.Category = inItem.Categories[0]
		}
		outFeed.Add(outItem)
	}

	var outFile *os.File
	if cfg.To == "-" {
		outFile = os.Stdout
	} else {
		var err error
		outFile, err = os.OpenFile(cfg.To, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, OutMode)
		if err != nil {
			log.Printf("Error opening output file '%s': %s", cfg.To, err)
			os.Exit(ec.IOErr)
		}
		defer func(outFile *os.File) {
			if err := outFile.Close(); err != nil {
				log.Printf("Error closing output file: %s", err)
				os.Exit(ec.IOErr)
			}
		}(outFile)
	}
	switch cfg.ToFmt {
	case FmtJson:
		if err := outFeed.WriteJSON(outFile); err != nil {
			if cfg.To == "-" {
				log.Printf("Error writing JSON feed to stdout: %s", err)
			} else {
				log.Printf("Error writing JSON feed '%s': %s", cfg.To, err)
			}
			os.Exit(ec.Failure)
		}
	case FmtRss:
		if err := outFeed.WriteRss(outFile); err != nil {
			if cfg.To == "-" {
				log.Printf("Error writing RSS feed to stdout: %s", err)
			} else {
				log.Printf("Error writing RSS feed '%s': %s", cfg.To, err)
			}
			os.Exit(ec.Failure)
		}
	case FmtAtom:
		if err := outFeed.WriteAtom(outFile); err != nil {
			if cfg.To == "-" {
				log.Printf("Error writing Atom feed to stdout: %s", err)
			} else {
				log.Printf("Error writing Atom feed '%s': %s", cfg.To, err)
			}
			os.Exit(ec.Failure)
		}
	default:
		log.Printf("Internal error: unknown output format '%s'", cfg.ToFmt)
		os.Exit(ec.Failure)
	}
}
