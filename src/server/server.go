package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"encoding/json"
)

func main() {
	addr := ":9000"

	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	fmt.Printf(`# Entrepot utility

Bound to %s

Routes:
- GET /shields/(releases|snapshots)/**.ex) => serve shield
- GET /pom/(releases|snapshots)/** => serve POM
`, addr)

	http.HandleFunc("/", handler)
	http.ListenAndServe(addr, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	path := r.URL.Path

	if method == "GET" && strings.Index(path, "/entrepot/shields/") == 0 {
		shield(w, r)
		return
	}

	if method == "GET" && strings.Index(path, "/entrepot/pom") == 0 {
		pom(w, r)
		return
	}

	// ---

	badRequest(w, fmt.Sprintf("Bad request: %s %s\n", method, path))
}

func latestVersion(
	category string,
	dependencyPath string) (version string, err error) {

	githubApiUrl := fmt.Sprintf("https://api.github.com/repos/zengularity/entrepot/contents/%s/%s", category, dependencyPath)

	githubResp, err1 := http.Get(githubApiUrl)

	if err1 != nil {
		return "", err1
	}

	// ---

	defer githubResp.Body.Close()

	dec := json.NewDecoder(githubResp.Body)
	_, err2 := dec.Token()

	if err2 != nil {
		return "", err2
	}

	// ---

	var latest string = "0"

	for dec.More() {
		var info interface{}

		err3 := dec.Decode(&info)

		if err3 != nil {
			return "", err3
		}

		// ---

		obj := info.(map[string]interface{})
		name := obj["name"].(string)

		if strings.Compare(latest, name) < 0 {
			latest = name
		}
	}

	_, err4 := dec.Token()

	if err4 != nil {
		return "", err4
	}

	// ---

	return latest, nil
}

func shield(w http.ResponseWriter, r *http.Request) {
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
	latest, err1 := latestVersion(category, dependencyPath)

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

	version := strings.Replace(latest, "-", "--", -1)
	color := "blue"

	if category == "releases" {
		color = "green"
	}

	shieldUrl := fmt.Sprintf("https://img.shields.io/badge/entrepot-%s-%s.%s", version, color, ext)

	shieldResp, err5 := http.Get(shieldUrl)
	
	if err5 != nil {
		writeError(w, err5)
		return
	}

	defer shieldResp.Body.Close()

	// Forward headers
	headers := w.Header()

	for name, vs := range shieldResp.Header {
		for _, v := range vs {
			headers.Add(name, v)
		}
	}

	// Forward body
	io.Copy(w, shieldResp.Body)
}

func pom(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	components := strings.Split(path, "/")

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
	latest, err1 := latestVersion(category, dependencyPath)

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

	groupId := strings.Join(components[4:len(components)-1], ".")
	artifactId := strings.Join(components[len(components)-1:], "")
	
	title := fmt.Sprintf("Entrepot - %s.%s", groupId, artifactId)
	pomUrl := fmt.Sprintf("https://raw.githubusercontent.com/zengularity/entrepot/master/%s/%s/%s/%s-%s.pom", category, dependencyPath, latest, artifactId, latest)

	// SBT
	sbtArtifact := fmt.Sprintf(" % \"%s\"", artifactId)
	underscore := strings.Index(artifactId, "_")
	scalaVer := "<none>"

	if underscore != -1 {
		scalaVer = artifactId[underscore+1:]
		sbtArtifact = fmt.Sprintf(
			" %%%% \"%s\"", artifactId[:underscore])
	}

	sbtDependency := fmt.Sprintf("\"%s\"%s %% \"%s\"",
		groupId, sbtArtifact, latest)

	// Headers
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/html")

	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <title>%s</title>

    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />

    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.1.0/css/bootstrap.min.css" integrity="sha384-9gVQ4dYFwwWSjIDZnLEWnxCjeSWFphJiwGPXr1jddIhOegiu1FwO5qRGvFXOdJZ4" crossorigin="anonymous" />

    <style type="text/css">
#scalaVersion, #scalaVersion::before {
  font-weight: bold
}

#scalaVersion::before {
  content: 'Scala binary version: '
}
    </style>
  </head>
  <body>
    <div class="container">
      <h1>%s</h1>

      <p>
        <a id="pomUrl" href="%s">POM URL</a><br />
        How to use Entrepot reposition: <a href="https://github.com/zengularity/entrepot/#usage">See documentation</a>
      </p>

      <hr />

      <div id="sbt" class="card">
        <h2 class="card-title">SBT</h2>
        <p id="scalaVersion" class="card-subtitle">%s</p>
        <pre class="card-text">%s</pre>
      </div>

      <div id="maven" class="card">
        <h2 class="card-title">Maven</h2>
        <pre class="card-text">&lt;dependency&gt;
    &lt;groupId&gt;%s&lt;/groupId&gt;
    &lt;artifactId&gt;%s&lt;/artifactId&gt;
    &lt;version&gt;%s&lt;/version&gt;
&lt;/dependency&gt;</pre>
      </div>
    </div>
  </body>
</html>`, title, title, pomUrl, scalaVer, sbtDependency,
		groupId, artifactId, latest)
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
