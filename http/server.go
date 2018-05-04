package http

import (
	"bytes"
	"fmt"
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
	router.StdGET("/*", h.Generic)
	router.StdPOST("/*", h.Generic)
	router.StdPUT("/*", h.Generic)
	router.StdDELETE("/*", h.Generic)
}

func (h *Handler) findEndpoint(method string, reqURL *url.URL, reqHeader http.Header) *stubserver.ConfigRequest {
	var (
		defaultEndpoint   *stubserver.ConfigRequest
		bestMatchEndpoint *stubserver.ConfigRequest
	)
	if len(h.cfg.Endpoints) > 0 {
		defaultEndpoint = &h.cfg.Endpoints[0]
	}

	for k, endpoint := range h.cfg.Endpoints {
		// Only check endpoints with the right method
		if !strings.EqualFold(method, endpoint.Method) {
			continue
		}

		// if our default is not with the method requested, let's update the default
		if defaultEndpoint.Method != endpoint.Method {
			defaultEndpoint = &h.cfg.Endpoints[k]
		}

		if strings.HasPrefix(endpoint.URL, "~") {
			endpointURL := strings.TrimSpace(endpoint.URL[1:])

			regex := regexp.MustCompile(endpointURL)
			if ok := regex.MatchString(reqURL.String()); ok {
				bestMatchEndpoint = &h.cfg.Endpoints[k]
				continue
			}
			h.DebugLogger.Printf("%s doesn't match regex, url requested %s", endpointURL, reqURL)

			continue
		}

		endpointURL, _ := url.ParseRequestURI(endpoint.URL)
		if !strings.EqualFold(reqURL.EscapedPath(), endpointURL.EscapedPath()) {
			continue
		}

		match := true

		reqQuery := reqURL.Query()
		for k, ev := range endpointURL.Query() {
			if rv, ok := reqQuery[k]; ok {
				if len(ev) > 0 && ev[0] != "" && !strings.EqualFold(ev[0], rv[0]) {
					h.DebugLogger.Printf("%s doesn't match query string '%s=%s', url requested %s", endpointURL, k, ev, reqURL)
					match = false
					break
				}
			} else {
				h.DebugLogger.Printf("%s doesn't match query string '%s=%s', url requested %s", endpointURL, k, ev, reqURL)
				match = false
				break
			}
		}

		if !match {
			continue
		}
		h.DebugLogger.Printf("%s match query string, url requested %s", endpointURL, reqURL)
		bestMatchEndpoint = &h.cfg.Endpoints[k]

		for k, ev := range endpoint.Headers {
			if rv, ok := reqHeader[k]; ok {
				if len(ev) > 0 && ev[0] != "" && !strings.EqualFold(ev[0], rv[0]) {
					h.DebugLogger.Printf("%s doesn't match header '%s=%s', url requested %s", endpointURL, k, ev, reqURL)
					match = false
					break
				}
			} else {
				h.DebugLogger.Printf("%s doesn't match header '%s=%s', url requested %s", endpointURL, k, ev, reqURL)
				match = false
				break
			}
		}

		if !match {
			continue
		}
		h.DebugLogger.Printf("%s match headers, url requested %s", endpointURL, reqURL)
		bestMatchEndpoint = &h.cfg.Endpoints[k]
	}

	if bestMatchEndpoint != nil {
		return bestMatchEndpoint
	}

	return defaultEndpoint
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

		io.Copy(w, body)
	}
}
