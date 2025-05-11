// In cmd/web/render.go
package main

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/justinas/nosurf" // Import for nosurf.Token
)

var bufferPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

// Modify render to accept r *http.Request
func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data *TemplateData) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("template %s does not exist", page)
		app.logger.Error("template does not exist", "template", page, "error", err)
		return err
	}

	// Add CSRF token to the data if data is not nil
	if data != nil {
		data.CSRFToken = nosurf.Token(r)
	}

	err := ts.Execute(buf, data)
	if err != nil {
		err = fmt.Errorf("failed to render template %s: %w", page, err)
		app.logger.Error("failed to render template", "template", page, "error", err)
		return err
	}

	w.WriteHeader(status)

	_, err = buf.WriteTo(w)
	if err != nil {
		err = fmt.Errorf("failed to write template to response: %w", err)
		app.logger.Error("failed to write template to response", "error", err)
		return err
	}

	return nil
}

// Modify renderPartial to accept r *http.Request
func (app *application) renderPartial(w http.ResponseWriter, r *http.Request, status int, page string, data any) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	templateName := filepath.Base(page)
	defineName := ""

	switch {
	case templateName == "habit_form.tmpl":
		defineName = "habit_form"
	case templateName == "habit_item.tmpl":
		defineName = "habit_item"
	case templateName == "habit_list.tmpl":
		defineName = "habit_list"
	default:
		defineName = templateName[:len(templateName)-len(filepath.Ext(templateName))]
	}

	// If the data passed to the partial is of type *TemplateData, add the CSRF token.
	// This is important if a partial itself is a form that will be POSTed (e.g., via HTMX replacing a form).
	if td, ok := data.(*TemplateData); ok {
		if td != nil { // Ensure td is not nil
			td.CSRFToken = nosurf.Token(r)
		}
	}

	// Use a more general template base if partials are not always in daily.tmpl context
	// For simplicity, assuming most partials are defined in files that also include base.tmpl logic
	// or are rendered into contexts where the main page's template set is available.
	// If you have truly standalone partials that are not part of any "page", this lookup might need adjustment.
	// For now, assuming daily.tmpl (or any page template) has all partials defined.
	ts, ok := app.templateCache["daily.tmpl"] // Or a base template that includes all partials definitions
	if !ok {
		// Fallback or error if daily.tmpl doesn't exist or isn't the right container
		// This might need a more robust way to find the template set containing the partial definition.
		// A common approach is to parse all partials into all page templates.
		err := fmt.Errorf("base template for partial %s (tried daily.tmpl) does not exist", page)
		app.logger.Error("base template for partial does not exist", "template", page, "error", err)
		return err
	}

	err := ts.ExecuteTemplate(buf, defineName, data)
	if err != nil {
		err = fmt.Errorf("failed to render partial template %s (defined as %s): %w", page, defineName, err)
		app.logger.Error("failed to render partial template", "template", page, "defineName", defineName, "error", err)
		return err
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)

	_, err = buf.WriteTo(w)
	if err != nil {
		err = fmt.Errorf("failed to write partial to response: %w", err)
		app.logger.Error("failed to write partial to response", "error", err)
		return err
	}

	return nil
}
