package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"hash/maphash"
	"html/template"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/net/publicsuffix"
	"net/http/cookiejar"

	"github.com/PuerkitoBio/goquery"
	"github.com/gobs/httpclient"
	"github.com/gobs/simplejson"
)

// https://{region}.craigslist.org/search[/area]/{category}?query={}&sort={}&hasPic=1&srchType=T&postedToday=1&bundleDuplicates=1&seach_distance={}&postal={}&min_price={}&max_price={}&crypto_currency=1&delivery_available=1

type ClClient struct {
	h *httpclient.HttpClient
}

func New(region Region) *ClClient {
	uri := fmt.Sprintf(searchuri, region)
	client := httpclient.NewHttpClient(uri)
	client.UserAgent = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36`
	client.AllowInsecure(true) // this is just to create a new transport with TLSClientConfig

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal("cannot create cookiejar: ", err)
	}
	client.SetCookieJar(jar)

	return &ClClient{h: client}
}

type Region string
type SubRegion string
type SortType string
type Category string

const (
	searchuri = "https://%v.craigslist.org/search/"

	SFBay        = Region("sfbay")
	SanFrancisco = SubRegion("sfa")
	SoutBay      = SubRegion("sby")
	EastBay      = SubRegion("eby")
	NortBay      = SubRegion("nby")
	Peninsula    = SubRegion("pen")
	SantCruz     = SubRegion("scz")

	PriceAsc  = SortType("priceasc")
	PriceDesc = SortType("pricedsc")
	Date      = SortType("date")
	Relevance = SortType("rel")

	ForSale     = Category("sss")
	Bikes       = Category("bia")
	Boats       = Category("boa")
	Cars        = Category("cta")
	Cellphones  = Category("moa")
	Computers   = Category("sya")
	Electronics = Category("ela")
	Free        = Category("zip")
	Music       = Category("msa")
	RVs         = Category("rva")
	Sporting    = Category("sga")
	Tools       = Category("tla")

	pageTemplate = `<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
    <meta charset="UTF-8">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/mini.css/3.0.1/mini-default.min.css">
    <style>
      .indent {
        padding-left: 12px;
      }
    </style>
  <head>
  <body>
    <h2>
      <a href="{{ .Url }}">{{ .Title }}</a>
      {{ if .Subtitle }}
        <small>({{ .Subtitle }})</small>
      {{ end }}
    </h2>

    <div class="container">
    {{ range .Entries }}
      <div class="row">
        <div class="col-sm-2">
          <a href="{{ .Href }}">
          {{ if .Image }}
            <img src="{{ .Image }}">
          {{ else }}
          <img src="data:image/png;base64,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" width="300" height="300" alt="No Image Available">
          {{ end }}
          </a>
        </div>

        <div class="col-sm-10">
          <h3>
            <a href="{{ .Href }}">{{ .Title }}</a>
            <small>Added: {{ .Datetime }}</small>
          </h3>
          <div class="indent">
          Price: {{ .Price }}<br/>
          {{ or .NearbyDesc .Neighborhood }}
          </div>
        </div>
      </div>
      {{ else }}
      No results
    {{ end }}
    </div>

  </body>
</html>
`
)

var hashSeed = maphash.MakeSeed()

type ResultEntry struct {
	Title        string
	Href         string
	Image        string
	Datetime     string
	Neighborhood string
	NearbyLoc    string
	NearbyDesc   string
	Price        string
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func (entry ResultEntry) Hash() uint64 {
	var h maphash.Hash

	h.SetSeed(hashSeed) // we need all hashes for the same string to be the same

	h.WriteString(normalize(entry.Title))
	h.WriteString(normalize(entry.Image))
	h.WriteString(normalize(entry.NearbyLoc))
	h.WriteString(normalize(entry.NearbyDesc))
	h.WriteString(normalize(entry.Neighborhood))
	h.WriteString(normalize(entry.Price))
	return h.Sum64()
}

type SearchResults struct {
	Title    string
	Subtitle string
	Url      string
	Entries  []ResultEntry
	Prev     string
	Next     string
}

type SearchOption func(params map[string]interface{})

func WithRegion(r Region) SearchOption {
	return func(params map[string]interface{}) {
		if string(r) != "" {
			params["region"] = string(r)
		}
	}
}

func WithSubregion(r SubRegion) SearchOption {
	return func(params map[string]interface{}) {
		if string(r) != "" {
			params["subregion"] = string(r)
		}
	}
}

func WithCategory(c Category) SearchOption {
	return func(params map[string]interface{}) {
		if string(c) != "" {
			params["category"] = string(c)
		}
	}
}

func By(by string) SearchOption {
	return func(params map[string]interface{}) {
		params["by"] = by
	}
}

func Query(q string) SearchOption {
	return func(params map[string]interface{}) {
		params["query"] = q
	}
}

func Sort(s SortType) SearchOption {
	return func(params map[string]interface{}) {
		if string(s) != "" {
			params["sort"] = string(s)
		}
	}
}

func Pictures(pictures bool) SearchOption {
	return func(params map[string]interface{}) {
		if pictures {
			params["hasPic"] = 1
		}
	}
}

func Today(today bool) SearchOption {
	return func(params map[string]interface{}) {
		if today {
			params["postedToday"] = 1
		}
	}
}

func Nearby(nearby bool) SearchOption {
	return func(params map[string]interface{}) {
		if nearby {
			params["searchNearby"] = 1
		}
	}
}

func Dedup(dedup bool) SearchOption {
	return func(params map[string]interface{}) {
		if dedup {
			params["bundleDuplicates"] = 1
		}
	}
}

func TitleOnly(only bool) SearchOption {
	return func(params map[string]interface{}) {
		if only {
			params["srchType"] = "T"
		}
	}
}

func SearchDistance(d int) SearchOption {
	return func(params map[string]interface{}) {
		params["search_distance"] = d
	}
}

func PostalCode(p string) SearchOption {
	return func(params map[string]interface{}) {
		params["postal_code"] = p
	}
}

func MinPrice(p int) SearchOption {
	return func(params map[string]interface{}) {
		if p > 0 {
			params["min_price"] = p
		}
	}
}

func MaxPrice(p int) SearchOption {
	return func(params map[string]interface{}) {
		if p > 0 {
			params["max_price"] = p
		}
	}
}

func (c *ClClient) Search(options ...SearchOption) (*SearchResults, error) {
	params := map[string]interface{}{}

	for _, opt := range options {
		opt(params)
	}

	reqs := []httpclient.RequestOption{}

	if r, ok := params["region"]; ok {
		uri := fmt.Sprintf(searchuri, r.(string))
		delete(params, "region")
		reqs = append(reqs, httpclient.URLString(uri))
	}

	path := ""
	cat := string(ForSale)

	if r, ok := params["subregion"]; ok {
		path = r.(string) + "/"
		delete(params, "subregion")
	}

	if c, ok := params["category"]; ok {
		cat = c.(string)
		delete(params, "category")
	}

	if by, ok := params["by"]; ok {
		delete(params, "by")

		switch by {
		case "owner":
			if strings.HasSuffix(cat, "a") {
				cat = cat[:len(cat)-1] + "o"
			}

		case "dealer":
			if strings.HasSuffix(cat, "d") {
				cat = cat[:len(cat)-1] + "d"
			}
		}
	}

	path += cat

	reqs = append(reqs, httpclient.Path(path))
	reqs = append(reqs, httpclient.Params(params))
	reqs = append(reqs, httpclient.Accept("*/*"))
	res, err := httpclient.CheckStatus(c.h.SendRequest(reqs...))
	results := SearchResults{Url: res.Response.Request.URL.String()}
	if err != nil {
		return &results, err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()

	if err != nil {
		return &results, err
	}

	if q, ok := params["query"]; ok {
		results.Title = q.(string)
	} else {
		results.Title = "Results"
	}

	dedup := params["bundleDuplicates"] != nil
	duplicates := map[uint64]bool{}

	doc.Find(".rows li.result-row").Each(func(i int, s *goquery.Selection) {
		title := s.Find(".result-heading a").First().Text()
		href, _ := s.Find(".result-heading a").First().Attr("href")
		iids, _ := s.Find("a.result-image").Attr("data-ids")
		datetime, _ := s.Find(".result-info .result-date").First().Attr("datetime")
		hood := s.Find(".result-meta .result-hood").First().Text()
		nearby := s.Find(".result-meta .nearby").First()
		loc, _ := nearby.Attr("title")
		ldesc := nearby.Text()
		price := s.Find(".result-meta .result-price").First().Text()

		image := ""
		ids := strings.Split(iids, ",")
		if len(iids) > 0 && len(ids) > 0 {
			parts := strings.Split(ids[0], ":")
			image = fmt.Sprintf("https://images.craigslist.org/%v_300x300.jpg", parts[1])
		}

		entry := ResultEntry{
			Title:        title,
			Href:         href,
			Image:        image,
			Datetime:     datetime,
			NearbyLoc:    loc,
			NearbyDesc:   strings.TrimSpace(ldesc),
			Neighborhood: strings.TrimSpace(hood),
			Price:        price,
		}

		if dedup {
			h := entry.Hash()
			if duplicates[h] == true {
				//log.Println("hash", h, "duplicate", entry)
				return
			}

			duplicates[h] = true
			//log.Println("duplicates", duplicates)
		}

		results.Entries = append(results.Entries, entry)

		//fmt.Println("<!-------------------------------------------------------------------------------->")
		//fmt.Println(goquery.OuterHtml(s))
		//fmt.Println("<!-------------------------------------------------------------------------------->")
	})

	results.Prev, _ = doc.Find(".buttons .prev").Attr("href")
	results.Next, _ = doc.Find(".buttons .next").Attr("href")

	return &results, nil
}

func mapCategory(name string) Category {
	var categories = map[string]Category{
		"all":         ForSale,
		"bikes":       Bikes,
		"boats":       Boats,
		"cars":        Cars,
		"phones":      Cellphones,
		"computers":   Computers,
		"electronics": Electronics,
		"free":        Free,
		"music":       Music,
		"rvs":         RVs,
		"sports":      Sporting,
		"tools":       Tools,
	}

	if c, ok := categories[name]; ok {
		return c
	}

	return Category(name)
}

func applyFilter(f string, in []ResultEntry) (out []ResultEntry) {
	if f == "" {
		return in
	}

	f = strings.ToLower(f)

	var any bool
	var fpat []string

	if strings.Contains(f, "|") { // any
		any = true
		fpat = strings.Split(f, "|")
	} else if strings.ContainsAny(f, "&, ") { // all
		fpat = regexp.MustCompile("[& ,]").Split(f, -1)
	} else { // one element
		any = true
		fpat = []string{f}
	}

	neg := map[string]bool{}
	mcount := 0
	ncount := 0

	for i, f := range fpat {
		if len(f) == 0 { // shouldn't happen but...
			log.Fatalf("empty filter in %q", fpat)
		}

		if strings.ContainsAny(f[:1], "-!^") {
			fv := f[1:]
			fpat[i] = fv
			neg[fv] = true
			ncount++
		} else {
			mcount++
		}
	}

	out = make([]ResultEntry, 0, len(in))
	re := regexp.MustCompile("(" + strings.Join(fpat, "|") + ")")

	for _, r := range in {
		res := re.FindAllString(strings.ToLower(r.Title), -1)
		if len(res) == 0 {
			if ncount == len(fpat) { // all negative
				out = append(out, r)
			}

			continue
		}

		// check that all positive pattern matched
		matches := 0
		nmatches := 0

		for _, v := range res {
			if neg[v] {
				nmatches++
			} else {
				matches++
			}
		}

		if nmatches > 0 {
			continue
		}

		if matches == mcount || // all positive matches
			(any && matches > 0) { // any positive match
			out = append(out, r)
		}
	}

	return
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func main() {
	region := flag.String("region", "sfbay", "Region")
	subregion := flag.String("subregion", "", "Subregion")
	cat := flag.String("cat", "sss", "Category")
	by := flag.String("by", "all", "all, owner, dealer")
	dedup := flag.Bool("dedup", true, "Bundle duplicates")
	pictures := flag.Bool("pictures", true, "Has pictures")
	sort := flag.String("sort", "", "Sort type (priceasc,pricedsc,date,rel)")
	titleOnly := flag.Bool("titles", false, "Search in title only")
	filter := flag.String("filter", "", "Title filter")
	today := flag.Bool("today", false, "Added today")
	min := flag.Int("min", 0, "Min price")
	max := flag.Int("max", 0, "Max price")
	html := flag.Bool("html", true, "Return an HTML page")
	browse := flag.Bool("browse", true, "Create HTML page and open browser")
	nearby := flag.Bool("nearby", false, "Search nearby")
	//url := flag.Bool("url", false, "Display Craigslist URL")

	debug := flag.Bool("debug", false, "Log HTTP requests")
	flag.Parse()

	if *debug {
		httpclient.StartLogging(false, false, true)
	}

	query := strings.Join(flag.Args(), " ")

	cl := New(Region(*region))
	res, err := cl.Search(
		WithSubregion(SubRegion(*subregion)),
		WithCategory(mapCategory(*cat)),
		By(*by),
		Dedup(*dedup),
		Pictures(*pictures),
		Sort(SortType(*sort)),
		TitleOnly(*titleOnly || *filter != ""),
		Today(*today),
		Nearby(*nearby),
		MinPrice(*min),
		MaxPrice(*max),
		Query(query))

	if err != nil {
		log.Fatalf("ERROR %v: %v", res.Url, err)
	}

	if *sort != "" {
		res.Subtitle = fmt.Sprintf("Sort: %v", *sort)
	}

	if *filter != "" {
		res.Subtitle = strings.TrimPrefix(fmt.Sprintf("%v, Filter Title: %v", res.Subtitle, *filter), ", ")
		res.Entries = applyFilter(*filter, res.Entries)
	}

	if *html && *browse {
		var b bytes.Buffer
		t := template.Must(template.New("webpage").Parse(pageTemplate))
		t.Execute(&b, res)

		// note that by default data: URLs don't "open" in MacOS
		// and you need to add a mapping scheme -> app
		// (see for example SwiftDefaultApps)
		durl := fmt.Sprintf("data:text/html;base64,%v", base64.StdEncoding.EncodeToString(b.Bytes()))
		openbrowser(durl)
	} else if *html {
		t := template.Must(template.New("webpage").Parse(pageTemplate))
		t.Execute(os.Stdout, res)
	} else {
		fmt.Println(simplejson.MustDumpString(res, simplejson.Indent(" ")))
	}
}
