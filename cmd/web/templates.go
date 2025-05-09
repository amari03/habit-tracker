package main

import (
	"html/template"
	"path/filepath"
)

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob("ui/html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		var ts *template.Template
		var parseErr error

		// For pages that are standalone (like login.tmpl and signup.tmpl), parse them directly.
		if name == "login.tmpl" || name == "signup.tmpl" || name == "landing.tmpl" { // standalone pages
			ts, parseErr = template.ParseFiles(page)
		} else {
			// Assume other pages use base.tmpl
			ts, parseErr = template.ParseFiles("ui/html/base.tmpl", page)
		}

		if parseErr != nil {
			return nil, parseErr
		}

		ts, parseErr = ts.ParseGlob("ui/html/partials/*.tmpl")
		if parseErr != nil {
			return nil, parseErr
		}

		cache[name] = ts
	}
	return cache, nil
}
