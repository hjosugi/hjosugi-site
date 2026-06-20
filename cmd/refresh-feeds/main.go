package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hjosugi/signal-garden/internal/ai"
	"github.com/hjosugi/signal-garden/internal/collector"
	"github.com/hjosugi/signal-garden/internal/config"
	"github.com/hjosugi/signal-garden/internal/feed"
	"github.com/hjosugi/signal-garden/internal/store"
	"github.com/hjosugi/signal-garden/internal/tagger"
)

func main() {
	app, err := config.LoadApp()
	if err != nil {
		fatal("load app config", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	feeds, err := config.LoadFeeds(app.FeedsPath)
	if err != nil {
		fatal("load feeds config", err)
	}
	if err := collector.ValidateFeeds(feeds); err != nil {
		fatal("validate feeds", err)
	}

	dataStore, err := store.OpenJSON(app.DataPath, app.MaxItems)
	if err != nil {
		fatal("open data store", err)
	}

	ollama := ai.NewOllama(app.OllamaBaseURL, app.OllamaChatModel, app.OllamaEmbedModel, app.SummaryLanguage, app.RequestTimeout*3)
	col := collector.New(
		feeds,
		feed.NewFetcher(app.RequestTimeout),
		dataStore,
		tagger.New(),
		ollama,
		app.EnableLLMSummary,
		app.EnableEmbeddings,
		app.FeedWorkers,
		logger,
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout(app, feeds))
	defer cancel()
	report, err := col.Refresh(ctx)
	if err != nil {
		fatal("refresh feeds", err)
	}
	fmt.Printf("refreshed feeds: added=%d updated=%d failed=%d total=%d\n", report.Added, report.Updated, report.Failed, dataStore.Count())
}

func refreshTimeout(app config.App, feeds []config.Feed) time.Duration {
	enabled := 0
	for _, cfg := range feeds {
		if cfg.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return app.RequestTimeout
	}
	workers := app.FeedWorkers
	if workers < 1 {
		workers = 1
	}
	batches := enabled / workers
	if enabled%workers != 0 {
		batches++
	}
	timeout := time.Duration(batches+1) * app.RequestTimeout * 2
	if timeout < 2*time.Minute {
		return 2 * time.Minute
	}
	return timeout
}

func fatal(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}
