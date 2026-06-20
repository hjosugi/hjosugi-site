package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hjosugi/signal-garden/internal/config"
	"github.com/hjosugi/signal-garden/internal/model"
	"github.com/hjosugi/signal-garden/internal/search"
	"github.com/hjosugi/signal-garden/internal/store"
)

type pageData struct {
	Site          config.Site
	Featured      []config.Project
	Others        []config.Project
	Items         []publicItem
	Facets        search.Facets
	EnabledFeeds  int
	GeneratedAt   time.Time
	GeneratedText string
	Year          int
	BaseURL       string
}

type publicItem struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"source_id"`
	SourceName  string    `json:"source_name"`
	SourceKind  string    `json:"source_kind"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Author      string    `json:"author,omitempty"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	CollectedAt time.Time `json:"collected_at"`
	Tags        []string  `json:"tags"`
}

type publicFeed struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Enabled bool     `json:"enabled"`
	Tags    []string `json:"tags"`
}

func main() {
	sitePath := flag.String("site", "config/site.json", "path to site config")
	feedsPath := flag.String("feeds", "config/feeds.json", "path to feeds config")
	dataPath := flag.String("data", "data/items.json", "path to collected items")
	outDir := flag.String("out", "public", "output directory")
	baseURL := flag.String("base-url", "", "absolute public base URL for sitemap")
	flag.Parse()

	site, err := config.LoadSite(*sitePath)
	if err != nil {
		fatal("load site config", err)
	}
	feeds, err := config.LoadFeeds(*feedsPath)
	if err != nil {
		fatal("load feeds config", err)
	}
	dataStore, err := store.OpenJSON(*dataPath, 100000)
	if err != nil {
		fatal("open data store", err)
	}

	items := dataStore.All()
	now := time.Now().UTC()
	data := pageData{
		Site:          site,
		Featured:      featuredProjects(site.Projects, true),
		Others:        featuredProjects(site.Projects, false),
		Items:         exportItems(items),
		Facets:        search.BuildFacets(items),
		EnabledFeeds:  enabledFeeds(feeds),
		GeneratedAt:   now,
		GeneratedText: now.Format("2006-01-02 15:04 UTC"),
		Year:          now.Year(),
		BaseURL:       strings.TrimRight(*baseURL, "/"),
	}

	if err := writeTemplate(filepath.Join(*outDir, "index.html"), homeTemplate, data); err != nil {
		fatal("write home page", err)
	}
	if err := writeTemplate(filepath.Join(*outDir, "signals", "index.html"), signalsTemplate, data); err != nil {
		fatal("write signals page", err)
	}
	if err := writeJSON(filepath.Join(*outDir, "data", "items.json"), data.Items); err != nil {
		fatal("write public items", err)
	}
	if err := writeJSON(filepath.Join(*outDir, "data", "site.json"), site); err != nil {
		fatal("write public site config", err)
	}
	if err := writeJSON(filepath.Join(*outDir, "data", "feeds.json"), exportFeeds(feeds)); err != nil {
		fatal("write public feeds", err)
	}
	if err := writeStaticAssets(*outDir); err != nil {
		fatal("write static assets", err)
	}
	if err := writeText(filepath.Join(*outDir, ".nojekyll"), ""); err != nil {
		fatal("write .nojekyll", err)
	}
	if err := writeText(filepath.Join(*outDir, "robots.txt"), robotsText(data.BaseURL)); err != nil {
		fatal("write robots.txt", err)
	}
	if data.BaseURL != "" {
		if err := writeText(filepath.Join(*outDir, "sitemap.xml"), sitemapXML(data.BaseURL)); err != nil {
			fatal("write sitemap.xml", err)
		}
	}

	fmt.Printf("exported static site: out=%s items=%d feeds=%d\n", *outDir, len(data.Items), data.EnabledFeeds)
}

func featuredProjects(projects []config.Project, featured bool) []config.Project {
	out := make([]config.Project, 0, len(projects))
	for _, project := range projects {
		if project.Featured == featured {
			out = append(out, project)
		}
	}
	return out
}

func enabledFeeds(feeds []config.Feed) int {
	count := 0
	for _, feed := range feeds {
		if feed.Enabled {
			count++
		}
	}
	return count
}

func exportItems(items []model.Item) []publicItem {
	out := make([]publicItem, 0, len(items))
	for _, item := range items {
		out = append(out, publicItem{
			ID:          item.ID,
			SourceID:    item.SourceID,
			SourceName:  item.SourceName,
			SourceKind:  item.SourceKind,
			Title:       item.Title,
			URL:         item.URL,
			Author:      item.Author,
			Summary:     item.Summary,
			Content:     truncate(item.Content, 1500),
			PublishedAt: item.PublishedAt,
			CollectedAt: item.CollectedAt,
			Tags:        append([]string(nil), item.Tags...),
		})
	}
	return out
}

func exportFeeds(feeds []config.Feed) []publicFeed {
	out := make([]publicFeed, 0, len(feeds))
	for _, feed := range feeds {
		out = append(out, publicFeed{
			ID:      feed.ID,
			Name:    feed.Name,
			Kind:    feed.Kind,
			Enabled: feed.Enabled,
			Tags:    append([]string(nil), feed.Tags...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max])) + "..."
}

func writeTemplate(path string, source string, data pageData) error {
	tmpl, err := template.New(filepath.Base(path)).Funcs(template.FuncMap{
		"join":  strings.Join,
		"lower": strings.ToLower,
	}).Parse(source)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeText(path, buf.String())
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeBytes(path, append(data, '\n'))
}

func writeStaticAssets(outDir string) error {
	css, err := os.ReadFile(filepath.Join("internal", "web", "static", "app.css"))
	if err != nil {
		return err
	}
	css = append(css, []byte(staticCSS)...)
	if err := writeBytes(filepath.Join(outDir, "static", "app.css"), css); err != nil {
		return err
	}
	if err := writeText(filepath.Join(outDir, "static", "app-static.js"), appStaticJS); err != nil {
		return err
	}
	return writeText(filepath.Join(outDir, "static", "static-signals.js"), signalsJS)
}

func writeText(path string, value string) error {
	return writeBytes(path, []byte(value))
}

func writeBytes(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func robotsText(baseURL string) string {
	text := "User-agent: *\nAllow: /\n"
	if baseURL != "" {
		text += "Sitemap: " + baseURL + "/sitemap.xml\n"
	}
	return text
}

func sitemapXML(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/</loc></url>
  <url><loc>%s/signals/</loc></url>
</urlset>
`, template.HTMLEscapeString(baseURL), template.HTMLEscapeString(baseURL))
}

func fatal(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}

const homeTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="dark">
  <meta name="description" content="{{.Site.Headline}}">
  <title>{{.Site.Handle}} - {{.Site.Headline}}</title>
  <link rel="stylesheet" href="static/app.css">
  <script defer src="static/app-static.js"></script>
</head>
<body class="crt">
<header class="site-header">
  <div class="shell nav-shell">
    <a class="brand" href="./" aria-label="Home"><span class="prompt">~/</span>{{.Site.Handle}}</a>
    <nav aria-label="Primary navigation">
      <a class="nav-link active" href="./">about</a>
      <a class="nav-link" href="signals/">signals</a>
      {{range .Site.Links}}<a class="nav-link" href="{{.URL}}" rel="me noopener" target="_blank">{{lower .Label}}</a>{{end}}
    </nav>
    <button class="ghost-button" id="crt-toggle" type="button" aria-pressed="true">crt:on</button>
  </div>
</header>
<main>
  <section class="shell hero">
    <div class="eyebrow"><span class="status-dot" aria-hidden="true"></span>{{.Site.Availability}}</div>
    <p class="terminal-line"><span class="prompt">$</span> whoami</p>
    <h1>{{.Site.DisplayName}}</h1>
    <p class="headline">{{.Site.Headline}}</p>
    <p class="lede">{{.Site.About}}</p>
    <div class="hero-actions">
      <a class="primary-button" href="#projects">view selected work</a>
      <a class="secondary-button" href="signals/">open signal radar</a>
    </div>
    <dl class="status-strip" aria-label="Site status">
      <div><dt>location</dt><dd>{{.Site.Location}}</dd></div>
      <div><dt>sources</dt><dd>{{.EnabledFeeds}} feeds</dd></div>
      <div><dt>indexed</dt><dd>{{len .Items}} items</dd></div>
      <div><dt>hosting</dt><dd>github pages</dd></div>
    </dl>
  </section>

  <section class="shell section" id="projects">
    <div class="section-heading">
      <div><p class="terminal-line"><span class="prompt">$</span> ls ./selected-work</p><h2>Selected work</h2></div>
      <p>Projects chosen for product impact, architecture decisions, and operational clarity.</p>
    </div>
    <div class="project-grid featured-grid">
      {{range .Featured}}
      <article class="project-card featured-card">
        <div class="card-topline"><span>featured</span><span>-&gt;</span></div>
        <h3><a href="{{.URL}}" target="_blank" rel="noopener">{{.Name}}</a></h3>
        <p>{{.Summary}}</p>
        <ul class="highlight-list">
          {{range .Highlights}}<li>{{.}}</li>{{end}}
        </ul>
        <div class="chip-row">{{range .Stack}}<span class="chip">{{.}}</span>{{end}}</div>
        <div class="card-links">
          <a href="{{.URL}}" target="_blank" rel="noopener">source</a>
          {{if .DemoURL}}<a href="{{.DemoURL}}" target="_blank" rel="noopener">live demo</a>{{end}}
        </div>
      </article>
      {{end}}
    </div>
    {{if .Others}}
    <div class="project-grid compact-grid">
      {{range .Others}}
      <article class="project-card compact-card">
        <h3><a href="{{.URL}}" target="_blank" rel="noopener">{{.Name}}</a></h3>
        <p>{{.Summary}}</p>
        <div class="chip-row">{{range .Stack}}<span class="chip">{{.}}</span>{{end}}</div>
      </article>
      {{end}}
    </div>
    {{end}}
  </section>

  <section class="shell section split-section">
    <div>
      <p class="terminal-line"><span class="prompt">$</span> cat skills.txt</p>
      <h2>Engineering focus</h2>
    </div>
    <div class="skill-groups">
      {{range .Site.Skills}}
      <div class="skill-group">
        <h3>{{.Name}}</h3>
        <p>{{join .Items " / "}}</p>
      </div>
      {{end}}
    </div>
  </section>

  <section class="shell section signal-callout">
    <div>
      <p class="terminal-line"><span class="prompt">$</span> ./collect --sources=high-signal</p>
      <h2>A public radar for technical change</h2>
      <p>The signal page is rebuilt from RSS, Atom, and YouTube feeds by GitHub Actions. It stays cheap because the deployed site is static.</p>
    </div>
    <a class="primary-button" href="signals/">open signals</a>
  </section>
</main>
<footer class="site-footer">
  <div class="shell footer-grid">
    <p><span class="prompt">$</span> static export from Go config and RSS data</p>
    <p>c {{.Year}} {{.Site.Handle}}</p>
  </div>
</footer>
</body>
</html>
`

const signalsTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="dark">
  <meta name="description" content="Search technical signals collected by {{.Site.Handle}}">
  <title>signals - {{.Site.Handle}}</title>
  <link rel="stylesheet" href="../static/app.css">
  <script defer src="../static/app-static.js"></script>
  <script defer src="../static/static-signals.js"></script>
</head>
<body class="crt">
<header class="site-header">
  <div class="shell nav-shell">
    <a class="brand" href="../" aria-label="Home"><span class="prompt">~/</span>{{.Site.Handle}}</a>
    <nav aria-label="Primary navigation">
      <a class="nav-link" href="../">about</a>
      <a class="nav-link active" href="./">signals</a>
      {{range .Site.Links}}<a class="nav-link" href="{{.URL}}" rel="me noopener" target="_blank">{{lower .Label}}</a>{{end}}
    </nav>
    <button class="ghost-button" id="crt-toggle" type="button" aria-pressed="true">crt:on</button>
  </div>
</header>
<main class="shell inbox-shell" data-signal-app data-items-url="../data/items.json">
  <section class="inbox-header">
    <div>
      <p class="terminal-line"><span class="prompt">$</span> signal-garden static search</p>
      <h1>Signal radar</h1>
      <p><span data-total-count>{{len .Items}}</span> indexed items / rebuilt {{.GeneratedText}} / press <kbd>/</kbd> to search</p>
    </div>
    <div class="runtime-panel">
      <span class="status-dot" aria-hidden="true"></span>
      <span>static</span>
      <span>/</span>
      <span>{{.EnabledFeeds}} feeds</span>
    </div>
  </section>

  <form class="search-form" id="static-search-form" role="search">
    <label for="signal-search">Search title, summary, content, source, and tags</label>
    <div class="search-row">
      <span class="search-prompt">&gt;</span>
      <input id="signal-search" name="q" placeholder="e.g. Spanner consistency, vector search, platform engineering" autocomplete="off">
      <button class="primary-button" type="submit">search</button>
    </div>
  </form>

  <div class="inbox-layout">
    <aside class="filter-panel">
      <div class="filter-section">
        <h2>tags</h2>
        <div data-tag-facets></div>
      </div>
      <div class="filter-section">
        <h2>sources</h2>
        <div data-source-facets></div>
      </div>
      <p class="static-note">This page has no server-side private mode. Anything deployed here is public.</p>
    </aside>

    <section class="results" aria-live="polite">
      <div class="results-topline">
        <p data-result-summary>Loading signals...</p>
        <a href="./" data-clear-filters hidden>clear filters</a>
      </div>
      <div data-results>
        <div class="empty-state">
          <p class="terminal-line"><span class="prompt">*</span> loading</p>
          <h2>Fetching the static signal index.</h2>
        </div>
      </div>
    </section>
  </div>
</main>
<footer class="site-footer">
  <div class="shell footer-grid">
    <p><span class="prompt">$</span> RSS data is refreshed by GitHub Actions</p>
    <p>c {{.Year}} {{.Site.Handle}}</p>
  </div>
</footer>
</body>
</html>
`

const staticCSS = `

.static-note {
  margin: 18px 0 0;
  color: var(--muted);
  font-size: 0.76rem;
}

.filter-section > div {
  display: grid;
  gap: 2px;
}

.result-hidden {
  display: none;
}
`

const appStaticJS = `(() => {
  const body = document.body;
  const toggle = document.getElementById("crt-toggle");
  const stored = localStorage.getItem("signal-garden-crt");
  if (stored === "off") {
    body.classList.add("crt-off");
    if (toggle) {
      toggle.textContent = "crt:off";
      toggle.setAttribute("aria-pressed", "false");
    }
  }

  toggle?.addEventListener("click", () => {
    const off = body.classList.toggle("crt-off");
    localStorage.setItem("signal-garden-crt", off ? "off" : "on");
    toggle.textContent = off ? "crt:off" : "crt:on";
    toggle.setAttribute("aria-pressed", String(!off));
  });

  document.addEventListener("keydown", (event) => {
    const target = event.target;
    const typing = target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement;
    if (event.key === "/" && !typing) {
      const search = document.getElementById("signal-search");
      if (search) {
        event.preventDefault();
        search.focus();
      }
    }
  });
})();
`

const signalsJS = `(() => {
  const app = document.querySelector("[data-signal-app]");
  if (!app) return;

  const form = document.getElementById("static-search-form");
  const input = document.getElementById("signal-search");
  const resultsNode = app.querySelector("[data-results]");
  const summaryNode = app.querySelector("[data-result-summary]");
  const clearNode = app.querySelector("[data-clear-filters]");
  const tagFacetsNode = app.querySelector("[data-tag-facets]");
  const sourceFacetsNode = app.querySelector("[data-source-facets]");
  const totalCountNode = app.querySelector("[data-total-count]");
  let items = [];

  const collator = new Intl.Collator(undefined, { sensitivity: "base" });

  fetch(app.dataset.itemsUrl)
    .then((response) => {
      if (!response.ok) throw new Error("items request failed");
      return response.json();
    })
    .then((data) => {
      items = Array.isArray(data) ? data : [];
      if (totalCountNode) totalCountNode.textContent = String(items.length);
      updateFromLocation();
    })
    .catch(() => {
      summaryNode.textContent = "Could not load the static signal index.";
      resultsNode.replaceChildren(emptyState("!", "The static data file is missing or unavailable."));
    });

  form?.addEventListener("submit", (event) => {
    event.preventDefault();
    const params = currentParams();
    params.set("q", input.value.trim());
    if (!params.get("q")) params.delete("q");
    navigate(params);
  });

  window.addEventListener("popstate", updateFromLocation);

  function updateFromLocation() {
    const params = currentParams();
    input.value = params.get("q") || "";
    render(params);
  }

  function render(params) {
    const query = params.get("q") || "";
    const tag = params.get("tag") || "";
    const source = params.get("source") || "";
    const ranked = rank(items, query)
      .filter((item) => !tag || item.tags?.some((value) => same(value, tag)))
      .filter((item) => !source || same(item.source_name, source) || same(item.source_id, source));
    const visible = ranked.slice(0, 80);

    summaryNode.textContent = summaryText(visible.length, ranked.length, query, tag, source);
    clearNode.hidden = !(query || tag || source);
    clearNode.onclick = (event) => {
      event.preventDefault();
      navigate(new URLSearchParams());
    };

    renderFacets(tagFacetsNode, facets(items, "tags"), "tag", tag, params);
    renderFacets(sourceFacetsNode, facets(items, "source_name"), "source", source, params);

    if (visible.length === 0) {
      resultsNode.replaceChildren(emptyState("!", "Try a broader query or clear the active filters."));
      return;
    }
    resultsNode.replaceChildren(...visible.map(renderCard));
  }

  function rank(values, query) {
    const terms = tokens(query);
    return [...values].map((item) => ({ item, score: score(item, terms, query) }))
      .sort((a, b) => b.score - a.score || dateValue(b.item) - dateValue(a.item))
      .map((entry) => entry.item);
  }

  function score(item, terms, query) {
    const dateScore = Math.max(0, dateValue(item) / 10000000000000);
    if (terms.length === 0) return dateScore;
    const title = norm(item.title);
    const tags = norm((item.tags || []).join(" "));
    const source = norm(item.source_name);
    const body = norm([item.summary, item.content, item.author].filter(Boolean).join(" "));
    let value = 0;
    for (const term of terms) {
      if (title === term) value += 12;
      if (title.includes(term)) value += 7;
      if (tags.includes(term)) value += 5;
      if (source.includes(term)) value += 3;
      if (body.includes(term)) value += 1;
    }
    const phrase = norm(query);
    if (phrase && title.includes(phrase)) value += 6;
    return value + dateScore;
  }

  function renderFacets(node, entries, key, activeValue, baseParams) {
    const all = facetLink("all", "", key, activeValue === "", baseParams, 0);
    const links = [all, ...entries.slice(0, 28).map((entry) => facetLink(entry.name, entry.name, key, same(activeValue, entry.name), baseParams, entry.count))];
    node.replaceChildren(...links);
  }

  function facetLink(label, value, key, active, baseParams, count) {
    const params = new URLSearchParams(baseParams);
    if (value) params.set(key, value);
    else params.delete(key);
    const link = document.createElement("a");
    link.className = "filter-link" + (active ? " active" : "");
    link.href = "?" + params.toString();
    link.addEventListener("click", (event) => {
      event.preventDefault();
      navigate(params);
    });

    const name = document.createElement("span");
    name.textContent = label;
    link.append(name);
    if (count > 0) {
      const badge = document.createElement("b");
      badge.textContent = String(count);
      link.append(badge);
    }
    return link;
  }

  function renderCard(item) {
    const article = document.createElement("article");
    article.className = "signal-card";

    const meta = document.createElement("div");
    meta.className = "signal-meta";
    meta.append(textSpan(item.source_name || "unknown source"));
    meta.append(textSpan(dateLabel(item)));
    if (item.source_kind) meta.append(textSpan(item.source_kind));

    const title = document.createElement("h2");
    const link = document.createElement("a");
    link.href = safeURL(item.url);
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.textContent = item.title || "Untitled";
    title.append(link);

    const summary = document.createElement("p");
    summary.textContent = item.summary || "No summary provided by the source.";

    const footer = document.createElement("div");
    footer.className = "signal-footer";
    const chips = document.createElement("div");
    chips.className = "chip-row";
    for (const tag of item.tags || []) {
      const chip = document.createElement("a");
      chip.className = "chip link-chip";
      const params = currentParams();
      params.set("tag", tag);
      chip.href = "?" + params.toString();
      chip.textContent = tag;
      chip.addEventListener("click", (event) => {
        event.preventDefault();
        navigate(params);
      });
      chips.append(chip);
    }
    footer.append(chips);
    if (item.author) {
      const author = document.createElement("span");
      author.textContent = "by " + item.author;
      footer.append(author);
    }

    article.append(meta, title, summary, footer);
    return article;
  }

  function emptyState(prefix, message) {
    const box = document.createElement("div");
    box.className = "empty-state";
    const line = document.createElement("p");
    line.className = "terminal-line";
    const prompt = document.createElement("span");
    prompt.className = "prompt";
    prompt.textContent = prefix;
    line.append(prompt, " no matching signals");
    const title = document.createElement("h2");
    title.textContent = message;
    box.append(line, title);
    return box;
  }

  function facets(values, field) {
    const counts = new Map();
    for (const item of values) {
      const entries = field === "tags" ? item.tags || [] : [item[field]].filter(Boolean);
      for (const entry of entries) counts.set(entry, (counts.get(entry) || 0) + 1);
    }
    return [...counts.entries()]
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count || collator.compare(a.name, b.name));
  }

  function summaryText(visible, total, query, tag, source) {
    const parts = [];
    if (query) parts.push("query \"" + query + "\"");
    if (tag) parts.push("tag " + tag);
    if (source) parts.push("source " + source);
    const scope = parts.length ? " for " + parts.join(", ") : "";
    return visible + " shown / " + total + " matches" + scope;
  }

  function navigate(params) {
    const query = params.toString();
    history.pushState(null, "", query ? "?" + query : location.pathname);
    updateFromLocation();
  }

  function currentParams() {
    return new URLSearchParams(location.search);
  }

  function tokens(value) {
    return norm(value).split(/[^a-z0-9_.+#-]+/).filter(Boolean);
  }

  function norm(value) {
    return String(value || "").trim().toLowerCase();
  }

  function same(a, b) {
    return norm(a) === norm(b);
  }

  function dateValue(item) {
    const value = Date.parse(item.published_at || item.collected_at || "");
    return Number.isFinite(value) ? value : 0;
  }

  function dateLabel(item) {
    const value = dateValue(item);
    if (!value) return "unknown";
    return new Date(value).toISOString().slice(0, 10);
  }

  function textSpan(value) {
    const span = document.createElement("span");
    span.textContent = value;
    return span;
  }

  function safeURL(value) {
    try {
      const url = new URL(value, location.href);
      return url.protocol === "http:" || url.protocol === "https:" ? url.href : "#";
    } catch (_) {
      return "#";
    }
  }
})();
`
