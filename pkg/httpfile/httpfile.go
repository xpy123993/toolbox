// Package httpfile includes a simple HTTP file server. Supports uploading.
package httpfile

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"time"

	_ "embed" // just for embedding
)

//go:embed assets/upload.html
var uploadHTML []byte

func noCacheHandler(h http.Handler, etagHeaders []string, noCacheHeaders map[string]string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// CreateHTTPServiceMux creates a HTTP mux for `directory`.
//
// `/download` to list files under `directory`.
// `/upload` is the file upload portal.
func CreateHTTPServiceMux(directory string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/download/", noCacheHandler(http.StripPrefix("/download", http.FileServer(http.Dir(directory))), []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}, map[string]string{
		"Expires":         time.Unix(0, 0).Format(time.RFC1123),
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}))
	mux.Handle("/", http.RedirectHandler("/download", http.StatusPermanentRedirect))
	mux.HandleFunc("/upload", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			rw.Write(uploadHTML)
			return
		}
		r.ParseMultipartForm(64 << 20)

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		defer file.Close()
		dst, err := os.Create(path.Join(directory, handler.Filename))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(rw, "OK")
	})
	return mux
}

// DownloadFileFromURL downloads from `URL` and saves to `LocalFileName`.
func DownloadFileFromURL(URL string, LocalFileName string) error {
	out, err := os.Create(LocalFileName)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// UploadFile uploads `LocalFileName` to `URL`.
func UploadFile(URL string, LocalFileName string) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("file", path.Base(LocalFileName))
	if err != nil {
		return err
	}

	// open file handle
	fh, err := os.Open(LocalFileName)
	if err != nil {
		return err
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(URL, contentType, bodyBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("returns error status code: %v in %v", resp.StatusCode, string(respBody))
	}
	return nil
}
