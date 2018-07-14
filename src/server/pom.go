package main

import (
	"fmt"
	"errors"
	"strings"
	"net/http"
	"text/template"
)

type Pom struct {
	Category string
	DependencyPath string
	Latest string
	GroupId string
	ArtifactId string
	ScalaVersion string
	SbtDependency string
	PomUrl string
	JavadocUrl string
	ShieldUrl string
}

func pom(w http.ResponseWriter, r *http.Request, ctx *Context) {
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

	groupId := strings.Join(components[4:len(components)-1], ".")
	artifactId := strings.Join(components[len(components)-1:], "")

	pomUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s/%s/%s/%s-%s.pom", *ctx.GithubUser, *ctx.GithubRepository, category, dependencyPath, latest, artifactId, latest)

	javadocUrl := fmt.Sprintf("//%s/%s/javadoc/%s/%s/%s!/index.html",
		r.Host, *ctx.GithubRepository, category, dependencyPath, latest)

	shieldUrl := shieldUrl(ctx, category, latest, "svg")

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

	pom := Pom {
		category, dependencyPath, latest, groupId, artifactId,
		scalaVer, sbtDependency, pomUrl, javadocUrl, shieldUrl }

	renderPom(w, r, pom, *ctx.PomTemplate)
}

func renderPom(
	w http.ResponseWriter,
	r *http.Request,
	pom Pom,
	templatePath string) {

	funcMap := template.FuncMap{
		"Title": strings.Title,
	}

	// Headers
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "text/html")

	tmpl, err1 := template.New("pom").
		Funcs(funcMap).ParseFiles(templatePath)

	if err1 != nil {
		writeError(w, err1)
		return
	}

	parsed := tmpl.Templates()

	if len(parsed) < 1 {
		writeError(w, errors.New("No parsed template"))
		return
	}

	// ---

	parsed[0].Execute(w, pom)
}
