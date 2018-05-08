package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
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

func templateBody(w http.ResponseWriter, req *http.Request, endpoint *stubserver.ConfigRequest, r io.Reader) io.Reader {
	if r == nil {
		return nil
	}

	b, _ := ioutil.ReadAll(r)
	tmpl := template.Must(template.New("body").Parse(string(b)))

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

	req.ParseForm()
	data["Request"] = req

	if req.Body != nil && req.Header.Get("Content-Type") == "application/json" {
		if body, err := ioutil.ReadAll(req.Body); err == nil {
			var j map[string]interface{}
			if err := json.Unmarshal(body, &j); err == nil {
				data["JSON"] = j
			}
		}
	}

	buf := new(bytes.Buffer)
	tmpl.Execute(buf, data)
	return buf
}

func extractHeader(r io.Reader) (int, http.Header, io.Reader) {
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	var statusCode int
	header := http.Header{}

	validHTTPProtocol := regexp.MustCompile(`^HTTP/[1-9](.[1-9])? ([245][0-9][0-9])`)

	body := bufio.NewReader(tee)
	for {
		line, _ := body.ReadString('\n')
		if line == "" || line == "\n" {
			// empty line means we read all headers
			break
		}

		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// it's a comment ignore it
			continue
		}

		if statusCode == 0 {
			if validHTTPProtocol.MatchString(line) {
				statusCode, _ = strconv.Atoi(validHTTPProtocol.FindStringSubmatch(line)[2])
			}
		} else {
			parts := strings.SplitN(line, ":", 2)
			header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	if statusCode == 0 {
		// No headers available let's return the original content
		return 0, nil, buf
	}

	return statusCode, header, body
}

func (h *Handler) Generic(w http.ResponseWriter, req *http.Request) {
	endpoint := h.findEndpoint(req.Method, req.URL, req.Header)
	if endpoint == nil {
		fdhttp.ResponseJSON(w, http.StatusNotFound, map[string]interface{}{
			"error":   "not_found",
			"message": "didn't match with any endpoint",
		})
		return
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

			var (
				statusCode int
				header     http.Header
			)

			statusCode, header, body = extractHeader(f)
			if statusCode != 0 {
				endpoint.Response.StatusCode = statusCode
			}
			if len(header) > 0 {
				endpoint.Response.Headers = header
			}
		} else {
			body = bytes.NewBufferString(endpoint.Response.Data)
		}

		for k, values := range endpoint.Response.Headers {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}

	w.WriteHeader(endpoint.Response.StatusCode)
	if body != nil {
		body = templateBody(w, req, endpoint, body)
		io.Copy(w, body)
	}
}
