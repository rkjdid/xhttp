package xhttp

import (
	"net/http"
	"path"
	"html/template"
	"log"
	"fmt"
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
			Status: http.StatusOK, // defaults to ok, some servers might not call wr.WriteHeader at all
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
	Name       string
	Handler    http.Handler
	HandleFunc func(http.ResponseWriter, *http.Request)
}

// ServeHTTP satisfies the http.Handler interface, in turns it tries for ls.Handler,
// ls.HandleFunc, or returns default http.NotFound.
func (ls *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = WrapCustomRW(w)
	var prefix string
	if ls.Name != "" {
		prefix = ls.Name + "> "
	}
	if ls.Handler != nil {
		ls.Handler.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}

	log.Printf("%s> serving %s -> %s (%d)",
		prefix, r.Header.Get("X-FORWARDED-FOR"), r.URL, w.(*CustomResponseWriter).Status)
}

