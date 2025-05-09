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

		// For pages that are standalone (like login.tmpl), parse them directly.
		// For others, parse them with base.tmpl.
		if name == "login.tmpl" { // Add other standalone pages here if any (e.g., "signup.tmpl" if it also shouldn't have nav)
			ts, parseErr = template.ParseFiles(page)
		} else {
			// Assume other pages use base.tmpl
			ts, parseErr = template.ParseFiles("ui/html/base.tmpl", page)
		}

		if parseErr != nil {
			return nil, parseErr
		}

		// After parsing the main page (either standalone or with base),
		// parse all partials into the template set.
		// This allows any page to call any partial it needs.
		ts, parseErr = ts.ParseGlob("ui/html/partials/*.tmpl")
		if parseErr != nil {
			return nil, parseErr
		}

		cache[name] = ts
	}
	return cache, nil
}