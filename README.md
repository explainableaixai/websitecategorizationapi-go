# websitecategorizationapi-go

Topic classification for web pages, from Go. Send an article URL or a domain to `Classify` and get back categories from the IAB content taxonomy with confidence scores. Publishers, ad servers and analytics pipelines use the answer for contextual targeting, brand safety and reporting. The service itself is documented at [website categorization for publishers and ad tech](https://www.websitecategorizationapi.com).

```bash
go get github.com/explainableaixai/websitecategorizationapi-go
```

## First request

```go
client := websitecategorizationapi.New(os.Getenv("AQ_API_KEY"))
res, err := client.Classify(ctx, "https://example.com/2026/09/transfer-window-review")
if err != nil {
	return err
}
cats, _ := res["categories"].([]any)
for _, x := range cats {
	c := x.(map[string]any)
	fmt.Printf("tier %v  %-30v %.2f\n", c["tier"], c["name"], c["confidence"])
}
```

In the documented reply, every entry of `categories` names its taxonomy node (`id` and `name`), its depth in the tree (`tier`) and how sure the model is (`confidence`). Timing and a request identifier sit under `meta`, so quote `meta.request_id` when you contact support. The client returns the decoded JSON as a `Result` (`map[string]any`) without reshaping it.

## Classifying at publish time

The cheapest moment to classify an article is once, when it is published. A CMS webhook handler in Go does it in a few lines:

```go
func onPublish(w http.ResponseWriter, r *http.Request) {
	var ev struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted) // reply fast; classify in background

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		res, err := client.Classify(ctx, ev.URL)
		if err != nil {
			log.Printf("classify %s: %v", ev.ID, err)
			return
		}
		saveCategories(ev.ID, res) // store with the article
	}()
}
```

Replying before the call finishes keeps the CMS responsive. The categories are then ready for the first ad request.

## From categories to ad-server keys

Ad servers accept key-values on the ad call. Two or three category IDs above a confidence floor make good keys:

```go
func topicKeys(res websitecategorizationapi.Result, floor float64, max int) []string {
	raw, _ := res["categories"].([]any)
	type kv struct {
		id   string
		conf float64
	}
	var picked []kv
	for _, x := range raw {
		c, _ := x.(map[string]any)
		conf, _ := c["confidence"].(float64)
		if conf >= floor {
			picked = append(picked, kv{fmt.Sprint(c["id"]), conf})
		}
	}
	sort.Slice(picked, func(i, j int) bool { return picked[i].conf > picked[j].conf })
	var out []string
	for i := 0; i < len(picked) && i < max; i++ {
		out = append(out, picked[i].id)
	}
	return out
}
```

`topicKeys(res, 0.5, 2)` is a reasonable starting point for targeting.

## Brand safety is a different threshold

Targeting can tolerate a wrong guess. Brand safety cannot. When you check for sensitive categories, look at every returned category, including low-confidence ones, and route borderline pages to review rather than straight to "safe". Keep the full response on file so a reviewer can see why a page was flagged.

## Pages that fight back

Some URLs give a classifier little to read: galleries, video pages with no transcript, paywalled articles, apps that render in the browser. If results look thin, classify the bare domain for a site-level label, or classify a text-heavy section page and use its categories for the thin pages beneath it.

## Client reference

- `New(key) *Client` sets the hosted `BaseURL` and a 30 second `HTTPClient`.
- `Classify(ctx, value) (Result, error)` sends one POST with `query`, `data_type=url` and your key.
- `*APIError` is returned for statuses of 400 and above, with `Status` and `Body`.
- Empty key or value returns an error before any request.
- Response bodies are read up to 2 MiB.

Handle failures by status:

```go
var apiErr *websitecategorizationapi.APIError
if errors.As(err, &apiErr) {
	if apiErr.Status == 429 {
		// retry later with backoff
	} else if apiErr.Status == 401 || apiErr.Status == 403 {
		// fix the key or top up the plan
	}
}
```

## Volume and cost

Classify each article once, and again only when its text changes. Page views do not need new calls. Sites with large back catalogues can classify old articles in a slow background job, a few requests at a time. For very large domain lists where only site-level labels matter, [bulk URL categories without per-call pricing](https://www.urlcategorizationdatabase.com) are usually cheaper than calling the API for each domain.

## Related data sets

- A topic taxonomy files AI products under technology. For [AI tool domains sorted by what each tool does](https://www.aitoolsblocklist.com), use the dedicated register.
- Security teams can [turn DNS exports into an AI tool inventory](https://www.shadowaitools.com) with the log-scanning service.
- For allow and block decisions rather than topics, use [content filtering categories for firewalls](https://www.webfilteringdatabase.com).

## Testing

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, `{"categories":[{"id":"483","name":"Sports","tier":1,"confidence":0.91}]}`)
}))
defer srv.Close()
client := websitecategorizationapi.New("test")
client.BaseURL = srv.URL
```

## Choosing a tier for reporting

Dashboards read best at tier 1: a dozen or two broad topics that anyone understands. Campaign planning often needs tier 2, where Sports splits into individual sports and Automotive into vehicle types. Deeper tiers help brand safety teams, who care about narrow sensitive topics. Since the full response is stored, one classification feeds all three views.

## Beyond Go

Node.js services use [the websitecategorization npm package](https://www.npmjs.com/package/websitecategorization). Rust has [a crate of the same name as this module](https://crates.io/crates/websitecategorizationapi), PHP has [a Composer library](https://packagist.org/packages/websitecategorization/websitecategorizationapi), and Flutter has [a pub.dev client](https://pub.dev/packages/websitecategorizationapi).

## License

MIT. Category identifiers follow the IAB Tech Lab taxonomy, whose names belong to their owner.
