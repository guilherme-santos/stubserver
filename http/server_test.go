package http_test

import (
	gohttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/guilherme-santos/stubserver"
	"github.com/guilherme-santos/stubserver/http"
	"github.com/stretchr/testify/assert"
)

func TestGeneric_NoEndpoint(t *testing.T) {
	var cfg stubserver.Config

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodGet, "http://stubserver:8080/test", nil)
	assert.NoError(t, err)

	h.Generic(w, req)
	assert.Equal(t, gohttp.StatusNotFound, w.Code)
}

func TestGeneric_SendStatusCode(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{
				Response: stubserver.ConfigResponse{
					StatusCode: gohttp.StatusCreated,
				},
			},
		},
	}

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodGet, "http://stubserver:8080/test", nil)
	assert.NoError(t, err)

	h.Generic(w, req)
	assert.Equal(t, gohttp.StatusCreated, w.Code)
}

func TestGeneric_SendData(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{
				Response: stubserver.ConfigResponse{
					Data:       "my-data",
					StatusCode: gohttp.StatusCreated,
				},
			},
		},
	}

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodGet, "http://stubserver:8080/test", nil)
	assert.NoError(t, err)

	h.Generic(w, req)
	assert.Equal(t, cfg.Endpoints[0].Response.Data, w.Body.String())
}

func TestGeneric_SendDataAsTemplate(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{
				Response: stubserver.ConfigResponse{
					Data:       `{{.Request.Method}} {{.Request.Host}} {{.Query.query}} {{index (index .Request.Header "Content-Type") 0}}`,
					StatusCode: gohttp.StatusCreated,
				},
			},
		},
	}

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodPut, "http://stubserver:8080/test?query=string", nil)
	req.Header.Add("Content-Type", "application/json")
	assert.NoError(t, err)

	h.Generic(w, req)
	assert.Equal(t, "PUT stubserver:8080 string application/json", w.Body.String())
}

func TestGeneric_SendDataAsFile(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{
				Response: stubserver.ConfigResponse{
					Data:       "@" + filepath.Join("testdata", "html.golden"),
					StatusCode: gohttp.StatusCreated,
				},
			},
		},
	}

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodGet, "http://stubserver:8080/test", nil)
	assert.NoError(t, err)

	h.Generic(w, req)
	assert.Equal(t, gohttp.StatusCreated, w.Code)
	assert.Len(t, w.HeaderMap, 0)
	assert.Equal(t, `<!DOCTYPE html>
<html>
    <body>
        <h1>My First Heading</h1>
        <p>My first paragraph.</p>
    </body>
</html>
`, w.Body.String())
}

func TestGeneric_SendDataAsFileWithHeaders(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{
				Response: stubserver.ConfigResponse{
					Data: "@" + filepath.Join("testdata", "html_with_header.golden"),
				},
			},
		},
	}

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodGet, "http://stubserver:8080/test", nil)
	assert.NoError(t, err)

	h.Generic(w, req)
	assert.Equal(t, gohttp.StatusCreated, w.Code)
	assert.Equal(t, "Mon, 07 May 2018 19:22:13 GMT", w.Header().Get("Date"))
	assert.Equal(t, "-1", w.Header().Get("Expires"))
	assert.Equal(t, "private, max-age=0", w.Header().Get("Cache-Control"))
	assert.Equal(t, "text/html; charset=ISO-8859-1", w.Header().Get("Content-Type"))
	assert.Equal(t, "gws", w.Header().Get("Server"))
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"))
	assert.Equal(t, `<!DOCTYPE html>
<html>
    <body>
        <h1>My First Heading</h1>
        <p>My first paragraph.</p>
    </body>
</html>
`, w.Body.String())
}

func TestGeneric_SendDataAsFileWithTemplate(t *testing.T) {
	cfg := stubserver.Config{
		Endpoints: []stubserver.ConfigRequest{
			{
				Response: stubserver.ConfigResponse{
					Data: "@" + filepath.Join("testdata", "html_with_template.golden"),
				},
			},
		},
	}

	h := http.NewHandler(cfg)

	w := httptest.NewRecorder()
	req, err := gohttp.NewRequest(gohttp.MethodGet, "http://stubserver:8080/test", nil)
	assert.NoError(t, err)
	req.Header.Add("Accept", "application/json")

	h.Generic(w, req)
	assert.Equal(t, gohttp.StatusCreated, w.Code)
	assert.Equal(t, "Mon, 07 May 2018 19:22:13 GMT", w.Header().Get("Date"))
	assert.Equal(t, "-1", w.Header().Get("Expires"))
	assert.Equal(t, "private, max-age=0", w.Header().Get("Cache-Control"))
	assert.Equal(t, "text/html; charset=ISO-8859-1", w.Header().Get("Content-Type"))
	assert.Equal(t, "gws", w.Header().Get("Server"))
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"))
	assert.Equal(t, `<!DOCTYPE html>
<html>
    <body>
        <h1>GET http://stubserver:8080/test</h1>
        <p>Headers</p>
        <p>Accept: application/json</p>
    </body>
</html>
`, w.Body.String())
}
