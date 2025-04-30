package main

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func (app *application) render(w http.ResponseWriter, status int, page string, data *TemplateData) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("template %s does not exist", page)
		app.logger.Error("template does not exist", "template", page, "error", err)
		return err
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

func (app *application) renderPartial(w http.ResponseWriter, status int, page string, data any) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	// Extract the template name from the path
	templateName := filepath.Base(page)
	defineName := ""

	// Determine the define name based on the template
	switch {
	case templateName == "habit_form.tmpl":
		defineName = "habit_form"
	case templateName == "habit_item.tmpl":
		defineName = "habit_item"
	case templateName == "habit_list.tmpl":
		defineName = "habit_list"
	default:
		// Default to using the filename without extension as the define name
		defineName = templateName[:len(templateName)-len(filepath.Ext(templateName))]
	}

	// Look up the template by its filename
	ts, ok := app.templateCache["daily.tmpl"] // Most partials are used in daily.tmpl
	if !ok {
		err := fmt.Errorf("template for partial %s does not exist", page)
		app.logger.Error("template for partial does not exist", "template", page, "error", err)
		return err
	}

	err := ts.ExecuteTemplate(buf, defineName, data)
	if err != nil {
		err = fmt.Errorf("failed to render partial template %s: %w", page, err)
		app.logger.Error("failed to render partial template", "template", page, "error", err)
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
