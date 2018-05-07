package http

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/foodora/go-ranger/fdhttp"
	"github.com/guilherme-santos/stubserver"
)

type Handler struct {
	cfg         stubserver.Config
	DebugLogger *log.Logger
}

func NewHandler(cfg stubserver.Config) *Handler {
	return &Handler{
		cfg:         cfg,
		DebugLogger: log.New(ioutil.Discard, "[handler] ", log.LstdFlags),
	}
}

func (h *Handler) Init(router *fdhttp.Router) {
	router.StdGET("/*anything", h.Generic)
	router.StdPOST("/*anything", h.Generic)
	router.StdPUT("/*anything", h.Generic)
	router.StdDELETE("/*anything", h.Generic)
}

func (h *Handler) findEndpoint(method string, reqURL *url.URL, reqHeader http.Header) *stubserver.ConfigRequest {
	var endpointFound = struct {
		PassedMethod      bool
		PassedURL         bool
		PassedQueryString bool
		PassedHeaders     bool
		Endpoint          *stubserver.ConfigRequest
	}{}

	if len(h.cfg.Endpoints) > 0 {
		endpointFound.Endpoint = &h.cfg.Endpoints[0]
	}

	for k, endpoint := range h.cfg.Endpoints {
		if !strings.EqualFold(method, endpoint.Method) {
			h.DebugLogger.Printf("%s %s: doesn't match method '%s' endpoint=%s", method, reqURL, endpoint.Method, endpoint.URL)
			continue
		}

		if !endpointFound.PassedMethod {
			h.DebugLogger.Printf("%s %s: match method '%s' endpoint=%s", method, reqURL, endpoint.Method, endpoint.URL)
			endpointFound.PassedMethod = true
			endpointFound.Endpoint = &h.cfg.Endpoints[k]
		}

		if strings.HasPrefix(endpoint.URL, "~") {
			endpointURL := strings.TrimSpace(endpoint.URL[1:])

			regex := regexp.MustCompile(endpointURL)
			if ok := regex.MatchString(reqURL.String()); ok {
				if !endpointFound.PassedURL {
					h.DebugLogger.Printf("%s %s: match regex '%s'", method, reqURL, endpoint.URL)
					endpointFound.PassedURL = true
					endpointFound.Endpoint = &h.cfg.Endpoints[k]
				}
			} else {
				h.DebugLogger.Printf("%s %s: doesn't match regex %s", method, reqURL, endpointURL)
			}

			continue
		} else {
			endpointURL, _ := url.ParseRequestURI(endpoint.URL)
			if !strings.EqualFold(reqURL.EscapedPath(), endpointURL.EscapedPath()) {
				continue
			}

			if !endpointFound.PassedURL {
				h.DebugLogger.Printf("%s %s: match URL '%s'", method, reqURL, endpoint.URL)
				endpointFound.PassedURL = true
				endpointFound.Endpoint = &h.cfg.Endpoints[k]
			}

			match := true

			reqQuery := reqURL.Query()
			for k, ev := range endpointURL.Query() {
				if rv, ok := reqQuery[k]; ok {
					if len(ev) > 0 && ev[0] != "" && !strings.EqualFold(ev[0], rv[0]) {
						h.DebugLogger.Printf("%s %s: doesn't match query string '%s=%s' endpoint=%s", method, reqURL, k, ev, endpointURL)
						match = false
						break
					}
				} else {
					h.DebugLogger.Printf("%s %s: doesn't match query string '%s=%s' endpoint=%s", method, reqURL, k, ev, endpointURL)
					match = false
					break
				}
			}

			if !match {
				continue
			}

			if !endpointFound.PassedQueryString && len(endpointURL.Query()) > 0 {
				h.DebugLogger.Printf("%s %s: match query string endpoint=%s", method, reqURL, endpointURL)
				endpointFound.PassedQueryString = true
				endpointFound.Endpoint = &h.cfg.Endpoints[k]
			}
		}

		match := true

		for k, ev := range endpoint.Headers {
			if rv, ok := reqHeader[k]; ok {
				if len(ev) > 0 && ev[0] != "" && !strings.EqualFold(ev[0], rv[0]) {
					h.DebugLogger.Printf("%s %s: doesn't match header '%s=%s' endpoint=%s", method, reqURL, k, ev, endpoint.URL)
					match = false
					break
				}
			} else {
				h.DebugLogger.Printf("%s %s: doesn't match header '%s=%s' endpoint=%s", method, reqURL, k, ev, endpoint.URL)
				match = false
				break
			}
		}

		if !match {
			continue
		}

		if !endpointFound.PassedHeaders {
			h.DebugLogger.Printf("%s %s: match header endpoint=%s", method, reqURL, endpoint.URL)
			endpointFound.PassedHeaders = true
			endpointFound.Endpoint = &h.cfg.Endpoints[k]
		}
	}

	return endpointFound.Endpoint
}

func templateBody(w http.ResponseWriter, req *http.Request, endpoint *stubserver.ConfigRequest, body io.Reader) io.Reader {
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)
	tmpl := template.Must(template.New("body").Parse(buf.String()))

	data := map[string]interface{}{}

	if strings.HasPrefix(endpoint.URL, "~") {
		endpointURL := strings.TrimSpace(endpoint.URL[1:])
		regex := regexp.MustCompile(endpointURL)
		params := regex.FindStringSubmatch(req.RequestURI)
		data["RouteParam"] = params[1:]
	} else {
		q := req.URL.Query()
		query := map[string]string{}
		for k, v := range q {
			query[k] = v[0]
		}
		data["Query"] = query
	}
	data["Request"] = req

	buf.Reset()
	tmpl.Execute(buf, data)
	return buf
}

func (h *Handler) Generic(w http.ResponseWriter, req *http.Request) {
	endpoint := h.findEndpoint(req.Method, req.URL, req.Header)
	if endpoint == nil {
		fdhttp.ResponseJSON(w, http.StatusNotFound, map[string]interface{}{
			"error":   "not_found",
			"message": "didn't match with any endpoint",
		})
	}

	var body io.Reader

	if endpoint.Response.Data != "" {
		if strings.HasPrefix(endpoint.Response.Data, "@") {
			// it's a file
			f, err := os.Open(endpoint.Response.Data[1:])
			if err != nil {
				fdhttp.ResponseJSON(w, http.StatusInternalServerError, map[string]interface{}{
					"error":   "invalid_response",
					"message": fmt.Sprintf("cannot open response file: %s", err),
				})
			}

			body = f
		} else {
			body = bytes.NewBufferString(endpoint.Response.Data)
		}

		for k, values := range endpoint.Response.Headers {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(endpoint.Response.StatusCode)

		body := templateBody(w, req, endpoint, body)
		io.Copy(w, body)
	}
}
