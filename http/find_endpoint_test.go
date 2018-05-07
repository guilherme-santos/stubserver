package http

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/guilherme-santos/stubserver"
	"github.com/stretchr/testify/assert"
)

func TestFindEndpoint_DefaultEndpoint(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users", Method: "PUT"},
			{URL: "/users", Method: "POST"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/default")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[0], *endpoint)
}

func TestFindEndpoint_DefaultMethodEndpoint(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users", Method: "GET"},
			{URL: "/users", Method: "POST"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/default")
	endpoint := h.findEndpoint("POST", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[1], *endpoint)
}

func TestFindEndpoint_BetterMatch(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users", Method: "GET"},
			{URL: "/users/1", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/1")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[1], *endpoint)
}

func TestFindEndpoint_WithoutQueryString(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users/1", Method: "GET"},
			{URL: "/users/1?field=address", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/1")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[0], *endpoint)
}

func TestFindEndpoint_WithQueryString(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users/1", Method: "GET"},
			{URL: "/users/1?field=address", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/1?field=address")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[1], *endpoint)
}

func TestFindEndpoint_WithNonMatchQueryString(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users/1", Method: "GET"},
			{URL: "/users/1?field=address", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/1?field=company")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[0], *endpoint)
}

func TestFindEndpoint_WithQueryStringWithoutValue(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users/1", Method: "GET"},
			{URL: "/users/1?field=", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/1?field=address")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[1], *endpoint)
}

func TestFindEndpoint_Header(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users/1", Method: "GET", Headers: http.Header{"X-Version": []string{"1.0.0"}}},
			{URL: "/users/1", Method: "GET", Headers: http.Header{"X-Version": []string{"2.0.0"}}},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/1")

	header := http.Header{}
	endpoint := h.findEndpoint("GET", reqURL, header)
	assert.Equal(t, cfg.Endpoints[0], *endpoint)

	header = http.Header{"X-Version": []string{"1.0.0"}}
	endpoint = h.findEndpoint("GET", reqURL, header)
	assert.Equal(t, cfg.Endpoints[0], *endpoint)

	header = http.Header{"X-Version": []string{"2.0.0"}}
	endpoint = h.findEndpoint("GET", reqURL, header)
	assert.Equal(t, cfg.Endpoints[1], *endpoint)
}

func TestFindEndpoint_RegexURLMatch(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users", Method: "GET"},
			{URL: "~/users/[1-9]", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/5")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[1], *endpoint)
}

func TestFindEndpoint_RegexURLDontMatch(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{URL: "/users", Method: "GET"},
			{URL: "~/users/[1-9]$", Method: "GET"},
		},
	}

	h := NewHandler(cfg)
	reqURL, _ := url.ParseRequestURI("/users/10")
	endpoint := h.findEndpoint("GET", reqURL, http.Header{})
	assert.Equal(t, cfg.Endpoints[0], *endpoint)
}
