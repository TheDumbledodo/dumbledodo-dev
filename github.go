package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type GHUser struct {
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
	CreatedAt   string `json:"created_at"`
	HTMLURL     string `json:"html_url"`
}

type GHRepo struct {
	Stars int  `json:"stars"`
	Fork  bool `json:"fork"`
}

type GHData struct {
	User  GHUser   `json:"user"`
	Repos []GHRepo `json:"repos"`
}

func (g GHData) Since() string {
	if t, err := time.Parse(time.RFC3339, g.User.CreatedAt); err == nil {
		return t.Format("2006")
	}
	return ""
}

type GHView struct {
	GHData
	Stars int
	Heat  HeatView
}

type ghStore struct {
	mu       sync.RWMutex
	data     GHData
	username string
	token    string
}

func newGHStore(snapshot fs.FS) *ghStore {
	s := &ghStore{username: os.Getenv("GITHUB_USER"), token: os.Getenv("GITHUB_TOKEN")}
	if s.username == "" {
		s.username = "TheDumbledodo"
	}
	if b, err := fs.ReadFile(snapshot, "github.json"); err == nil {
		_ = json.Unmarshal(b, &s.data)
	}
	return s
}

func (s *ghStore) start() {
	go func() {
		for {
			if err := s.refresh(); err != nil {
				log.Printf("github refresh: %v (using cached data)", err)
			}
			time.Sleep(6 * time.Hour)
		}
	}()
}

func (s *ghStore) get(path string, v any) error {
	req, _ := http.NewRequest("GET", "https://api.github.com"+path, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "dumbledodo.dev")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	c := &http.Client{Timeout: 8 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (s *ghStore) refresh() error {
	var u GHUser
	if err := s.get("/users/"+s.username, &u); err != nil {
		return err
	}
	var raw []struct {
		Stars int  `json:"stargazers_count"`
		Fork  bool `json:"fork"`
	}
	if err := s.get("/users/"+s.username+"/repos?per_page=100&sort=pushed", &raw); err != nil {
		return err
	}
	d := GHData{User: u}
	for _, r := range raw {
		d.Repos = append(d.Repos, GHRepo{r.Stars, r.Fork})
	}
	s.mu.Lock()
	s.data = d
	s.mu.Unlock()
	return nil
}

func (s *ghStore) view() GHView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := GHView{GHData: s.data}
	for _, r := range s.data.Repos {
		if !r.Fork {
			v.Stars += r.Stars
		}
	}
	return v
}
