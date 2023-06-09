package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	TEMPLATE = "template"
	HTML     = "html"
)

type Item struct {
	name     string
	url      string
	fullPath string
	typeFile string
	dataFile string
}

func main() {
	var sitePath string
	var staticFolder string
	var staticPath string
	var port string
	var indexHtml string
	var indexTemplate string
	var dataJson string
	flag.StringVar(&sitePath, "path", ".", "Basic site path")
	flag.StringVar(&staticFolder, "staticd", "", "Public folder for serve static files")
	flag.StringVar(&staticPath, "staticp", "/static/", "Url to serve static files")
	flag.StringVar(&port, "port", "8080", "Port to serve site")
	flag.StringVar(&indexHtml, "html", "index.html", "File name for html file")
	flag.StringVar(&indexTemplate, "template", "index.tmpl", "File name for template file")
	flag.StringVar(&dataJson, "data", "data.json", "File name for data file")

	flag.Parse()

	structure := make(map[string]Item)
	err := filepath.Walk(sitePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(sitePath, path)
			fileName := filepath.Base(relPath)
			dirPath := filepath.Dir(relPath)
			if dirPath == "." {
				dirPath = ""
			}

			item, exists := structure[dirPath]

			if !exists {
				item = Item{
					url: "/" + dirPath,
				}
			}

			if fileName == indexHtml || fileName == indexTemplate {
				item.name = fileName
				item.fullPath = path

				if fileName == indexTemplate {
					item.typeFile = TEMPLATE
				} else {
					item.typeFile = HTML
				}
			}

			if fileName == dataJson {
				item.dataFile = path
			}

			structure[dirPath] = item
		}

		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", "./site", err)
		return
	}

	handleFunc := func(item Item) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			if item.typeFile == TEMPLATE {
				err := renderTemplate(item, w)
				if err != nil {
					renderError(err, w)
				}
				return
			}

			err := renderHtml(item, w, r)
			if err != nil {
				renderError(err, w)
			}
		}
	}

	for _, v := range structure {
		http.HandleFunc(v.url, handleFunc(v))
	}

	if staticFolder != "" {
		if !strings.HasSuffix(staticPath, "/") {
			staticPath += "/"
		}

		http.Handle(staticPath, http.StripPrefix(strings.TrimRight(staticPath, "/"), http.FileServer(http.Dir(staticFolder))))
	}

	http.ListenAndServe(":"+port, nil)
}

func renderTemplate(item Item, w http.ResponseWriter) error {
	tmpl, err := template.ParseFiles(item.fullPath)

	if err != nil {
		return err
	}

	dataContent, err := os.ReadFile(item.dataFile)
	if err != nil {
		return err
	}

	jsonContent := map[string]interface{}{}
	err = json.Unmarshal(dataContent, &jsonContent)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, item.name, jsonContent)
}

func renderHtml(item Item, w http.ResponseWriter, r *http.Request) error {
	data, err := os.ReadFile(item.fullPath)
	if err != nil {
		return err
	}
	w.Header().Add("Content-Type", "text/html")
	http.ServeContent(w, r, item.name, time.Now(), bytes.NewReader(data))

	return nil
}

func renderError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}
