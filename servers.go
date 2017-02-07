package xhttp

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
)

// CustomResponseWriter allows to store current status code of ResponseWriter.
type CustomResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (w *CustomResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *CustomResponseWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

func (w *CustomResponseWriter) WriteHeader(statusCode int) {
	// set w.Status then forward to inner ResposeWriter
	w.Status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func WrapCustomRW(wr http.ResponseWriter) http.ResponseWriter {
	if _, ok := wr.(*CustomResponseWriter); !ok {
		return &CustomResponseWriter{
			ResponseWriter: wr,
			Status:         http.StatusOK, // defaults to ok, some servers might not call wr.WriteHeader at all
		}
	}
	return wr
}

// HtmlServer is a simple html/template server helper
type HtmlServer struct {
	Root  string
	Name  string
	Data  interface{}
	Debug bool
}

func (hs *HtmlServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("dbg: %s, %s, %s", r.URL.Path, r.RequestURI, hs.Name)
	htmlPath := path.Join(hs.Root, hs.Name)
	t, err := template.ParseFiles(htmlPath)
	if err != nil {
		log.Printf("%s -> err parsing %s: %s", r.URL.Path, hs.Name, err)
		if hs.Debug {
			http.Error(w, fmt.Sprintf("in template.ParseFiles of %s: %s", hs.Name, err), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = t.ExecuteTemplate(w, hs.Name, hs.Data)
	if err != nil {
		log.Printf("%s -> err executing template %s: %s", r.URL.Path, hs.Name, err)
		if hs.Debug {
			http.Error(w, fmt.Sprintf("in template.Execute: %s", err), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
}

// WatServer is basically a 404 fallback server
type WatServer struct{}

func (ws *WatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "<html><head><title>what?</title></head><body>looking for <em>%s</em> ?</body>", r.URL.Path)
}

// LogServer is a simple log wrapper to either a http.Handler, or if Handler is nil,
// to HandleFunc. It provides simple logging before responding with one of the inner handlers
type LogServer struct {
	http.Handler
	Name       string
	HandleFunc func(http.ResponseWriter, *http.Request)
}

// ServeHTTP satisfies the http.Handler interface, in turns it tries for ls.Handler,
// ls.HandleFunc, or returns default http.NotFound.
func (ls *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = WrapCustomRW(w)
	if ls.Handler != nil {
		ls.Handler.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}

	log.Printf("%s> @%s -> %s (%d)", ls.Name,
		r.Header.Get("X-FORWARDED-FOR"), r.URL, w.(*CustomResponseWriter).Status)
}

// SiphonServer is useful to allow all patterns to redirect to the siphon url, /%s/
type SiphonServer struct {
	http.Handler
	Target string
}

func (ss *SiphonServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI != ss.Target {
		http.Redirect(w, r, ss.Target, http.StatusFound)
		return
	}
	ss.Handler.ServeHTTP(w, r)
	return
}
