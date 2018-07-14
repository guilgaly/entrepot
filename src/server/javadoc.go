package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"errors"
	"mime"
	"net/http"
	"os"
	"strings"
	"archive/zip"
	"time"
)

func javadoc(w http.ResponseWriter, r *http.Request, ctx *Context) {
	path := r.URL.Path
	req := strings.SplitAfter(path, "!")

	if len(req) != 2 {
		badRequest(w, fmt.Sprintf(
			"Invalid javadoc request: %s\n", path))
		return
	}

	// ---

	depPath := req[0]
	components := strings.Split(depPath[:len(depPath)-1], "/")
	l := len(components)

	if l != 8 {
		badRequest(w, fmt.Sprintf("Invalid javadoc path: %s\n", path))
		return
	}

	// ---

	basePath := strings.Join(components[3:8], "/")
	artifact := strings.Join(components[6:8], "-")

	javadocUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s/%s-javadoc.jar", *ctx.GithubUser, *ctx.GithubRepository, basePath, artifact)

	javadocResp, err1 := http.Get(javadocUrl)
	
	if err1 != nil {
		writeError(w, err1)
		return
	}

	if javadocResp.StatusCode != 200 {
		forward(w, javadocResp)
		return
	}

	jarSize := javadocResp.ContentLength

	if jarSize <= 0 {
		writeError(w, errors.New(
			fmt.Sprintf("Invalid javadoc size: %s", jarSize)))
		return
	}

	// ---

	buf, err2 := ioutil.TempFile("", *ctx.GithubRepository)

	if err2 != nil {
		writeError(w, err2)
		return
	}

	defer buf.Close()
	defer os.Remove(buf.Name())

	// ---

	if _, err := io.Copy(buf, javadocResp.Body); err != nil {
		writeError(w, err)
		return
	}

	// ---
	
	jarReader, err2 := zip.NewReader(buf, jarSize)

	if err2 != nil {
		writeError(w, err2)
		return
	}

	// ---

	innerPath := req[1][1:]
	var innerFile *zip.File = nil

	for _, f := range jarReader.File {
		if f.Name == innerPath {
			innerFile = f
		}
	}

	if innerFile == nil {
		notFound(w, fmt.Sprintf(
			"Path '%s' not found in Javadoc jar", innerPath))
		return
	}

	// ---

	mod := innerFile.FileHeader.Modified
	
	if ifMod := r.Header["If-Modified-Since"]; len(ifMod) > 0 {
		ifModT, err3 := time.Parse(time.RFC1123, ifMod[0])

		if err3 != nil {
			writeError(w, err3)
			return
		}

		// ---

		if !mod.After(ifModT) {
			w.WriteHeader(304)
			return
		}
	}

	// ---

	innerReader, err4 := innerFile.Open()

	if err4 != nil {
		writeError(w, err4)
		return
	}

	// ---

	extIdx := strings.LastIndex(innerPath, ".")
	mimeType := "text/plain"

	if extIdx > 0 {
		mimeType = mime.TypeByExtension(innerPath[extIdx:])
	}

	// Prepare headers
	headers := w.Header()

	headers.Add("Content-Type", mimeType)
	headers.Add("Last-Modified", mod.UTC().Format(time.RFC1123))

	// Output the inner content
	io.Copy(w, innerReader)
}
