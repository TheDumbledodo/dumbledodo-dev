package main

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var avatarURLPrefix = "https://cdn.builtbybit.com/avatars/"

type avatarImg struct {
	b    []byte
	kind string
}

type avatarStore struct {
	mu sync.RWMutex
	m  map[string]avatarImg
}

var avatars = &avatarStore{m: map[string]avatarImg{}}

func (s *avatarStore) get(uid string) (avatarImg, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[uid]
	return v, ok
}

func (s *avatarStore) set(uid string, v avatarImg) {
	s.mu.Lock()
	s.m[uid] = v
	s.mu.Unlock()
}

var okImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true}

func fetchAvatars(reviews []Review, have func(uid string) bool) {
	client := &http.Client{Timeout: 10 * time.Second}
	for _, r := range reviews {
		if r.UID == "" || r.Avatar == "" || have(r.UID) || !strings.HasPrefix(r.Avatar, avatarURLPrefix) {
			continue
		}
		v, err := downloadImage(client, r.Avatar)
		if err != nil {
			log.Printf("avatar %s: %v", r.UID, err)
			continue
		}
		avatars.set(r.UID, v)
	}
}

func downloadImage(c *http.Client, u string) (avatarImg, error) {
	resp, err := c.Get(u)
	if err != nil {
		return avatarImg{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return avatarImg{}, &httpStatusError{resp.StatusCode, ""}
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return avatarImg{}, err
	}
	kind := http.DetectContentType(b)
	if !okImageTypes[kind] {
		return avatarImg{}, &httpStatusError{-1, kind}
	}
	return avatarImg{b, kind}, nil
}

type httpStatusError struct {
	code int
	kind string
}

func (e *httpStatusError) Error() string {
	if e.code < 0 {
		return "not an image (got " + e.kind + ")"
	}
	return "status " + http.StatusText(e.code)
}

func (a *app) avatar(w http.ResponseWriter, r *http.Request) {
	v, ok := avatars.get(r.PathValue("uid"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", v.kind)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(v.b)
}
