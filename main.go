package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*.html static data
var assets embed.FS

var assetVer = func() string {
	h := fnv.New32a()
	for _, n := range []string{"static/site.css", "static/site.js"} {
		b, _ := assets.ReadFile(n)
		_, _ = h.Write(b)
	}
	return strconv.FormatUint(uint64(h.Sum32()), 36)
}()

type Link struct{ Label, URL string }

type SkillGroup struct {
	Group string
	Items []string
}

type Profile struct {
	Name, Tagline, Email, Discord string
	OpenTo                        string `json:"open_to"`
	Links                         []Link
	Skills                        []SkillGroup
}

type Group struct {
	ID, Title, Period string
}

type Project struct {
	N        string   `json:"n"`
	G        string   `json:"g"`
	D        string   `json:"d"`
	C        int      `json:"c"`
	T        int      `json:"t"`
	Featured bool     `json:"featured"`
	I        []string `json:"i"`
	Pin      bool     `json:"pin"`
	Mods     []string `json:"mods"`
	K        string   `json:"k"`
}

func (p Project) Role() string {
	if p.K == "fork" {
		return "Fork"
	}
	if p.T == 0 {
		return ""
	}
	share := float64(p.C) / float64(p.T)
	switch {
	case share >= 0.6:
		return "Author"
	case share >= 0.2:
		return "Co-developer"
	default:
		return "Contributor"
	}
}

func (p Project) Thumb() string {
	if len(p.I) > 0 {
		return p.I[0]
	}
	return ""
}

func (p Project) ImgN() int { return len(p.I) }

type Showcase struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Kicker  string   `json:"kicker"`
	Period  string   `json:"period"`
	Text    []string `json:"text"`
	Points  []string `json:"points"`
	Images  []string `json:"images"`
	Tags    []string `json:"tags"`
	Link    string   `json:"link"`
	LinkLbl string   `json:"link_label"`
	Logo    string   `json:"logo"`
}

type Review struct {
	Featured bool
	Quote    string
	Work     string
	Name     string `json:"name"`
	UID      string `json:"uid"`
	Avatar   string `json:"avatar"`
}

type Reviews struct {
	TotalReviews int `json:"total_reviews"`
	URL          string
	Items        []Review
}

type GalleryItem struct{ Slug, Caption, Group, Date string }

type Event struct {
	Name   string
	Videos []string
}

type Media struct {
	Gallery []GalleryItem
	Events  []Event
}

type Site struct {
	Profile  Profile
	Groups   []Group
	Projects []Project
	Reviews  Reviews
	Media    Media
	Showcase []Showcase
	Heat     HeatView
	gh       *ghStore
	Totals   Totals
}

type Totals struct{ Projects int }

func readJSON(fsys fs.FS, name string, v any) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		log.Fatalf("read %s: %v", name, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		log.Fatalf("parse %s: %v", name, err)
	}
}

func loadSite() *Site {
	var dataFS fs.FS
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		dataFS = os.DirFS(dir)
	} else {
		sub, err := fs.Sub(assets, "data")
		if err != nil {
			log.Fatal(err)
		}
		dataFS = sub
	}
	s := &Site{}
	readJSON(dataFS, "profile.json", &s.Profile)
	var pj struct {
		Groups []Group
		Items  []Project
	}
	readJSON(dataFS, "projects.json", &pj)
	s.Groups, s.Projects = pj.Groups, pj.Items
	readJSON(dataFS, "reviews.json", &s.Reviews)
	readJSON(dataFS, "media.json", &s.Media)
	readJSON(dataFS, "showcase.json", &s.Showcase)
	var days DaysData
	readJSON(dataFS, "days.json", &days)
	s.Heat = buildHeat(days)
	s.gh = newGHStore(dataFS)
	s.Totals.Projects = len(s.Projects)
	return s
}

type ProjectGroup struct {
	Group
	Items []Project
}

type ProjectsView struct {
	Groups []ProjectGroup
}

func (s *Site) projectsView() ProjectsView {
	var v ProjectsView
	for _, g := range s.Groups {
		var part []Project
		for _, p := range s.Projects {
			if p.G == g.ID {
				part = append(part, p)
			}
		}
		sort.SliceStable(part, func(i, j int) bool {
			if part[i].Pin != part[j].Pin {
				return part[i].Pin
			}
			if part[i].Featured != part[j].Featured {
				return part[i].Featured
			}
			return part[i].C > part[j].C
		})
		if len(part) > 0 {
			v.Groups = append(v.Groups, ProjectGroup{Group: g, Items: part})
		}
	}
	return v
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type GalleryView struct {
	Items []GalleryItem
	All   bool
	Total int
}

const galleryPreview = 14

func (s *Site) galleryView(all bool) GalleryView {
	v := GalleryView{All: all, Total: len(s.Media.Gallery)}
	for _, it := range s.Media.Gallery {
		if all || len(v.Items) < galleryPreview {
			v.Items = append(v.Items, it)
		}
	}
	return v
}

type Page struct {
	*Site
	View        ProjectsView
	ReviewsView ReviewsView
	Gallery     GalleryView
	GH          GHView
	Year        int
}

type ReviewsView struct {
	Reviews
	All   bool
	Shown []Review
}

const reviewsPreview = 6

func (s *Site) reviewsView(all bool) ReviewsView {
	v := ReviewsView{Reviews: s.Reviews, All: all}
	for _, r := range s.Reviews.Items {
		if all || (r.Featured && len(v.Shown) < reviewsPreview) {
			v.Shown = append(v.Shown, r)
		}
	}
	return v
}

type ModalView struct {
	GalleryItem
	Prev, Next string
	Pos, Count int
	Project    string
}

var icons = map[string]template.HTML{}

type app struct {
	site *Site
	tpl  *template.Template
}

func newApp() *app {
	site := loadSite()
	for _, n := range []string{"discord", "github", "youtube", "linkedin", "mail", "link"} {
		if b, err := fs.ReadFile(assets, "static/icons/"+n+".svg"); err == nil {
			icons[n] = template.HTML(strings.Replace(string(b), "<svg ", `<svg aria-hidden="true" focusable="false" `, 1))
		}
	}
	if ents, err := fs.ReadDir(assets, "static/icons"); err == nil {
		for _, e := range ents {
			if n := strings.TrimSuffix(e.Name(), ".svg"); strings.HasPrefix(n, "sk-") {
				if b, err := fs.ReadFile(assets, "static/icons/"+e.Name()); err == nil {
					icons[n] = template.HTML(strings.Replace(string(b), "<svg ", `<svg aria-hidden="true" focusable="false" `, 1))
				}
			}
		}
	}
	funcs := template.FuncMap{
		"skill":  func(name string) template.HTML { return icons[skillIcons[name]] },
		"skname": func(name string) string { return skillIcons[name] },
		"ver":    func() string { return assetVer },
		"rich":   rich,
		"avatar": func(uid string) string {
			if uid == "" {
				return ""
			}
			if _, ok := avatars.get(uid); ok {
				return "/avatars/" + uid
			}
			if _, err := fs.Stat(assets, "static/img/avatars/"+uid+".jpg"); err == nil {
				return "/static/img/avatars/" + uid + ".jpg"
			}
			return ""
		},
		"yt":    func(id string) string { return "https://www.youtube.com/watch?v=" + id },
		"ytimg": func(id string) string { return "https://img.youtube.com/vi/" + id + "/hqdefault.jpg" },
		"add":   func(a, b int) int { return a + b },
		"mod":   func(a, b int) int { return a % b },
		"iconFor": func(label string) string {
			switch strings.ToLower(label) {
			case "github":
				return "github"
			case "linkedin":
				return "linkedin"
			}
			return "link"
		},
		"qs":   func(s string) string { return url.QueryEscape(s) },
		"icon": func(name string) template.HTML { return icons[name] },
		"initials": func(name string) string {
			var out string
			for _, f := range strings.Fields(name) {
				out += strings.ToUpper(f[:1])
			}
			return out
		},
	}
	tpl := template.Must(template.New("").Funcs(funcs).ParseFS(assets, "templates/*.html"))
	go fetchAvatars(site.Reviews.Items, func(string) bool { return false })
	return &app{site: site, tpl: tpl}
}

func (a *app) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := a.tpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (a *app) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	a.render(w, "index", Page{
		Site:        a.site,
		View:        a.site.projectsView(),
		ReviewsView: a.site.reviewsView(false),
		Gallery:     a.site.galleryView(false),
		GH:          a.ghView(),
		Year:        time.Now().Year(),
	})
}

func (a *app) reviews(w http.ResponseWriter, r *http.Request) {
	a.render(w, "reviews", a.site.reviewsView(r.URL.Query().Get("all") == "1"))
}

func (a *app) ghView() GHView {
	v := a.site.gh.view()
	v.Heat = a.site.Heat
	return v
}

func (a *app) github(w http.ResponseWriter, _ *http.Request) {
	a.render(w, "github", a.ghView())
}

func (a *app) galleryList(w http.ResponseWriter, r *http.Request) {
	a.render(w, "gallery", a.site.galleryView(r.URL.Query().Get("all") == "1"))
}

func (a *app) gallery(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	bySlug := map[string]GalleryItem{}
	for _, it := range a.site.Media.Gallery {
		bySlug[it.Slug] = it
	}
	it, ok := bySlug[slug]
	if !ok {
		http.NotFound(w, r)
		return
	}

	list := make([]string, 0, len(a.site.Media.Gallery))
	proj := r.URL.Query().Get("proj")
	if proj != "" {
		for _, p := range a.site.Projects {
			if p.N == proj {
				list = append(list, p.I...)
			}
		}
	}

	caseID := r.URL.Query().Get("case")
	if proj == "" && caseID != "" {
		for _, c := range a.site.Showcase {
			if c.ID == caseID {
				list = append(list, c.Images...)
				proj = c.Title
			}
		}
	}
	if len(list) == 0 || !contains(list, slug) {
		proj, caseID = "", ""
		list = list[:0]
		for _, g := range a.site.Media.Gallery {
			list = append(list, g.Slug)
		}
	}
	idx := 0
	for i, s := range list {
		if s == slug {
			idx = i
		}
	}
	mk := func(s string) string {
		u := "/gallery/" + s
		if caseID != "" {
			u += "?case=" + url.QueryEscape(caseID)
		} else if proj != "" {
			u += "?proj=" + url.QueryEscape(proj)
		}
		return u
	}
	m := ModalView{GalleryItem: it, Pos: idx + 1, Count: len(list), Project: proj,
		Prev: mk(list[(idx+len(list)-1)%len(list)]), Next: mk(list[(idx+1)%len(list)])}
	a.render(w, "modal", m)
}

func (a *app) close(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }

func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https://img.youtube.com data:; font-src 'self'; style-src 'self'; style-src-attr 'unsafe-inline'; script-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func cache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		next.ServeHTTP(w, r)
	})
}

func main() {
	a := newApp()
	a.site.gh.start()
	mux := http.NewServeMux()
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /static/", cache(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))))
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(assets, "static/favicon.ico")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("GET /avatars/{uid}", a.avatar)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("GET /reviews", a.reviews)
	mux.HandleFunc("GET /github", a.github)
	mux.HandleFunc("GET /gallery", a.galleryList)
	mux.HandleFunc("GET /gallery/{slug}", a.gallery)
	mux.HandleFunc("GET /close", a.close)
	mux.HandleFunc("GET /{$}", a.index)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           secure(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

var linkRe = regexp.MustCompile(`\[([^]]+)]\((https://[^)\s]+)\)`)

func rich(s string) template.HTML {
	esc := template.HTMLEscapeString(s)
	return template.HTML(linkRe.ReplaceAllString(esc, `<a class="ext" href="$2" rel="noopener" target="_blank">$1</a>`))
}

var skillIcons = map[string]string{
	"Java": "sk-java", "Go": "sk-go", "Rust": "sk-rust", "TypeScript": "sk-typescript",
	"Spring Boot": "sk-spring", "Spring Security": "sk-spring", "Spring Data / JPA": "sk-spring", "Hibernate": "sk-hibernate", "REST APIs": "sk-api", "Vue": "sk-vue", "Next.js": "sk-nextjs",
	"MySQL": "sk-mysql", "PostgreSQL": "sk-postgresql", "MongoDB": "sk-mongodb",
	"Git": "sk-git", "Linux": "sk-linux", "Gradle": "sk-gradle", "Maven": "sk-maven",
}
