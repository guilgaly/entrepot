package main

import (
	"fmt"
	"flag"
	"io"
	"errors"
	"net/http"
	"os"
	"strings"
	"encoding/json"
	"text/template"
)

type Context struct {
	Address *string
	GithubUser *string
	GithubRepository *string
	PomTemplate *string
}

func main() {
	ctx := Context {
		flag.String("bind", ":9000", "address to bind"),
		flag.String("githubUser", "zengularity",
			"GitHub user name (or organization)"),
		flag.String("githubRepository", "entrepot",
			"GitHub repository (not prefixed by the user)"),
		flag.String("pomTemplate", "./resources/pom.html.tmpl",
			"Path to template for the POM display") }

	flag.Parse()

	if !flag.Parsed() {
		os.Exit(1)
		return
	}

	// ---

	addr := *ctx.Address

	tmpl, err1 := template.New("started").Parse(`
# GitHub as Maven repository

Bound to {{.Address}}

Routes:

  GET /{{.GithubRepository}}/shields/(releases|snapshots)/{a_group}/{an_artifact}.{ext}
  => Serve informational shields indicating the latest version
     for the managed dependencies.

  GET /{{.GithubRepository}}/pom/(releases|snapshots)/{a_group}/{an_artifact}
  => Serve HTML guide about how to use the specified dependency.

  GET /{{.GithubRepository}}/javadoc/(releases|snapshots)/{a_group}/{an_artifact}/{version}!/path/inside/javadoc-jar/file.ext
  => Serve the contents from the Javadoc JAR

  Placeholders:

  - a_group: Maven groupId, with / as separator (not .)
  - an_artifact: Maven artifactId, ended with _{scalaBinary} 
    for the Scala dependencies (e.g. benji-core_2.12)
  - ext: File extension for the shield image (e.g. svg, png)

`)

	if err1 == nil {
		tmpl.Execute(os.Stdout, ctx)
	}

	http.HandleFunc("/", handler(&ctx))
	http.ListenAndServe(addr, nil)
}

func handler(ctx *Context) func(http.ResponseWriter, *http.Request) {
	shieldsPrefix := fmt.Sprintf("/%s/shields/", *ctx.GithubRepository)
	pomPrefix := fmt.Sprintf("/%s/pom/", *ctx.GithubRepository)
	javadocPrefix := fmt.Sprintf("/%s/javadoc/", *ctx.GithubRepository)

	return func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.Path
		
		if method == "GET" && strings.Index(path, shieldsPrefix) == 0 {
			shield(w, r, ctx)
			return
		}
		
		if method == "GET" && strings.Index(path, pomPrefix) == 0 {
			pom(w, r, ctx)
			return
		}
		
		if method == "GET" && strings.Index(path, javadocPrefix) == 0 {
			javadoc(w, r, ctx)
			return
		}

		// ---
		
		badRequest(w, fmt.Sprintf("Bad request: %s %s\n", method, path))
	}
}

func shield(w http.ResponseWriter, r *http.Request, ctx *Context) {
	path := r.URL.Path
	dot := strings.LastIndex(path, ".")

	if dot == -1 {
		badRequest(w, fmt.Sprintf("Invalid dependency path, file extension expected: %s\n", path))
		return
	}

	// ---

	resource := path[:dot]
	ext := path[dot+1:]
	components := strings.Split(resource, "/")

	if len(components) < 6 {
		badRequest(w, fmt.Sprintf("Invalid dependency path: %s\n", path))
		return
	}

	// ---

	category := components[3]

	if category != "releases" && category != "snapshots" {
		badRequest(w, fmt.Sprintf("Invalid category, expect 'releases' or 'snapshots': %s\n", category))
		return
	}

	// ---

	dependencyPath := strings.Join(components[4:], "/")
	latest, err1 := latestVersion(ctx, category, dependencyPath)

	if err1 != nil {
		writeError(w, err1)
		return
	}

	if latest == "0" {
		notFound(w, fmt.Sprintf("Dependency not found: %s/%s",
			category, dependencyPath))
		return
	}

	// ---

	shieldResp, err5 := http.Get(shieldUrl(ctx, category, latest, ext))
	
	if err5 != nil {
		writeError(w, err5)
		return
	}

	defer shieldResp.Body.Close()

	forward(w, shieldResp)
}

func shieldUrl(
	ctx *Context,
	category string,
	latest string,
	ext string) string {

	version := strings.Replace(latest, "-", "--", -1)
	color := "blue"

	if category == "releases" {
		color = "green"
	}

	return fmt.Sprintf("https://img.shields.io/badge/%s-%s-%s.%s",
		*ctx.GithubRepository, version, color, ext)
}

func latestVersion(
	ctx *Context,
	category string,
	dependencyPath string) (version string, err error) {

	githubApiUrl := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s/%s",
		*ctx.GithubUser, *ctx.GithubRepository,
		category, dependencyPath)

	githubResp, err1 := http.Get(githubApiUrl)

	if err1 != nil {
		return "", err1
	}

	// ---

	defer githubResp.Body.Close()

	dec := json.NewDecoder(githubResp.Body)
	if _, err := dec.Token(); err != nil {
		return "", err
	}

	// ---

	var latest string = "0"

	for dec.More() {
		var info interface{}

		if err := dec.Decode(&info); err != nil {
			return "", err
		}

		// ---

		obj, ok := info.(map[string]interface{})

		if !ok {
			return "",
			errors.New("JSON value is not expected object")
		}

		// ---

		name := obj["name"].(string)

		if strings.Compare(latest, name) < 0 {
			latest = name
		}
	}

	if _, err := dec.Token(); err != nil {
		return "", err
	}

	// ---

	return latest, nil
}

func forward(w http.ResponseWriter, resp* http.Response) {
	// Forward headers
	headers := w.Header()

	for name, vs := range resp.Header {
		for _, v := range vs {
			headers.Add(name, v)
		}
	}

	// Forward body
	if _, err := io.Copy(w, resp.Body); err != nil {
		writeError(w, err)
		return
	}
}

func notFound(w http.ResponseWriter, msg string) {
	w.WriteHeader(404)
	w.Header().Add("Content-Type", "text/plain")

	fmt.Fprintf(w, msg)
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	w.Header().Add("Content-Type", "text/plain")
	
	fmt.Fprintf(w, err.Error())
}

func badRequest(w http.ResponseWriter, msg string) {
	w.WriteHeader(400)
	w.Header().Add("Content-Type", "text/plain")
	
	fmt.Fprintf(w, msg)
}
