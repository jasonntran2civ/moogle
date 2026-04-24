// Package guidelines scrapes USPSTF + NICE + AHRQ HTML pages with a
// minimal Colly-style crawler (spec §5.1.12). Per-source crawl rules
// declared inline below; refactor into config/guidelines-rules.yaml when
// adding a fourth source.
package guidelines

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/evidencelens/evidencelens/ingest/pkg/ingestcommon"
	"github.com/evidencelens/evidencelens/ingest/pkg/pubsubpub"
	"github.com/evidencelens/evidencelens/ingest/pkg/r2"
	"github.com/evidencelens/evidencelens/ingest/pkg/watermark"
)

type sourceRule struct {
	Name      string
	Seed      string
	LinkRegex *regexp.Regexp
}

var rules = []sourceRule{
	{
		Name:      "uspstf",
		Seed:      "https://www.uspreventiveservicestaskforce.org/uspstf/topic_search_results?topic_status=P",
		LinkRegex: regexp.MustCompile(`href="(/uspstf/recommendation/[^"#?]+)"`),
	},
	{
		Name:      "nice",
		Seed:      "https://www.nice.org.uk/guidance/published?type=apg,csg,cg,mpg,ph,sg,sc,ng",
		LinkRegex: regexp.MustCompile(`href="(/guidance/[A-Za-z0-9-]+)"`),
	},
	{
		Name:      "ahrq",
		Seed:      "https://www.ahrq.gov/research/findings/evidence-based-reports/index.html",
		LinkRegex: regexp.MustCompile(`href="(/research/findings/evidence-based-reports/[^"#?]+\.html)"`),
	},
}

type Ingester struct {
	logger   *slog.Logger
	wm       *watermark.Store
	archiver *r2.Archiver
	pub      *pubsubpub.Publisher
	fetcher  *ingestcommon.Fetcher
}

func New(logger *slog.Logger, wm *watermark.Store, arch *r2.Archiver, pub *pubsubpub.Publisher) *Ingester {
	return &Ingester{
		logger: logger, wm: wm, archiver: arch, pub: pub,
		fetcher: ingestcommon.NewFetcher(1, 2, "EvidenceLens-Guidelines/0.1 (mailto:contact@example.com)"),
	}
}

func (i *Ingester) Run(ctx context.Context) (ingestcommon.RunResult, error) {
	if err := i.wm.MarkRunning(ctx, "guidelines"); err != nil {
		return ingestcommon.RunResult{}, err
	}
	var counters ingestcommon.Counters

	for _, r := range rules {
		if ctx.Err() != nil {
			break
		}
		seedBody, err := i.fetcher.Get(ctx, r.Seed, nil)
		if err != nil {
			i.logger.Warn("seed fetch", "source", r.Name, "err", err)
			continue
		}
		base, _ := url.Parse(r.Seed)
		matches := r.LinkRegex.FindAllStringSubmatch(string(seedBody), -1)
		for _, m := range matches {
			if len(m) < 2 || ctx.Err() != nil {
				break
			}
			absoluteURL := resolveURL(base, m[1])
			if absoluteURL == "" {
				continue
			}
			counters.Fetched.Add(1)
			docID := r.Name + ":" + sha1Hex(absoluteURL)
			pageBody, err := i.fetcher.Get(ctx, absoluteURL, nil)
			if err != nil {
				counters.Failed.Add(1)
				continue
			}
			text := stripHTML(string(pageBody))
			payload, _ := json.Marshal(map[string]any{
				"source":     r.Name,
				"url":        absoluteURL,
				"text":       text,
				"fetched_at": time.Now().UTC().Format(time.RFC3339),
			})
			source := "guideline-" + r.Name
			key, err := i.archiver.Put(ctx, source, docID, payload)
			if err != nil {
				counters.Failed.Add(1)
				continue
			}
			counters.Archived.Add(1)
			if _, err := i.pub.PublishRaw(ctx, source, docID, key); err == nil {
				counters.Published.Add(1)
			}
		}
	}

	stamp := time.Now().UTC().Format("2006-01-02")
	_ = i.wm.Set(ctx, "guidelines", stamp, "idle", "")
	return ingestcommon.RunResult{
		DocsFetched:   counters.Fetched.Load(),
		DocsArchived:  counters.Archived.Load(),
		DocsPublished: counters.Published.Load(),
		HighWatermark: stamp,
	}, nil
}

func resolveURL(base *url.URL, ref string) string {
	rel, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return base.ResolveReference(rel).String()
}

func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

var (
	scriptRE = regexp.MustCompile(`(?is)<script.*?</script>`)
	styleRE  = regexp.MustCompile(`(?is)<style.*?</style>`)
	tagRE    = regexp.MustCompile(`<[^>]+>`)
	wsRE     = regexp.MustCompile(`\s+`)
)

func stripHTML(s string) string {
	s = scriptRE.ReplaceAllString(s, " ")
	s = styleRE.ReplaceAllString(s, " ")
	s = tagRE.ReplaceAllString(s, " ")
	s = wsRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
