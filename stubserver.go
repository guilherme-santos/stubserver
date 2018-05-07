package stubserver

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Endpoints []ConfigRequest
}

type ConfigRequest struct {
	URL     string
	Method  string
	Headers http.Header
	// Response can be string or ConfigResponse
	// see MarshalYAML to more details
	Response ConfigResponse
}

type ConfigResponse struct {
	Headers    http.Header
	StatusCode int
	Data       string
}

// UnmarshalYAML need to map to a totally different struct to be able receive the format
// that we expect.
func (c *ConfigRequest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	hack := struct {
		URL      string
		Method   string
		Headers  yaml.MapSlice
		Response ConfigResponse
	}{}

	if err := unmarshal(&hack); err != nil {
		return err
	}

	hack.URL = strings.TrimSpace(hack.URL)
	// if is not regex validate URL
	if !strings.HasPrefix(hack.URL, "~") {
		_, err := url.ParseRequestURI(hack.URL)
		if err != nil {
			return fmt.Errorf("url '%s' is not valid: %s", c.URL, err)
		}
	}
	c.URL = hack.URL
	c.Method = hack.Method
	c.Headers = http.Header{}
	c.Response = hack.Response

	return mapSliceToHeader(hack.Headers, c.Headers)
}

func (r *ConfigResponse) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var data string

	if err := unmarshal(&data); err == nil {
		r.StatusCode = http.StatusOK
		r.Headers = http.Header{}
		r.Data = data
		return nil
	}

	hack := struct {
		Headers    yaml.MapSlice
		StatusCode int
		Data       string
	}{}

	if err := unmarshal(&hack); err != nil {
		return err
	}

	r.Headers = http.Header{}
	r.StatusCode = hack.StatusCode
	r.Data = hack.Data

	return mapSliceToHeader(hack.Headers, r.Headers)
}

func mapSliceToHeader(mapSlice yaml.MapSlice, header http.Header) error {
	for _, kv := range mapSlice {
		key := kv.Key.(string)

		if str, ok := kv.Value.(string); ok {
			header.Set(key, str)
			continue
		}
		if i, ok := kv.Value.(int); ok {
			header.Set(key, strconv.Itoa(i))
			continue
		}

		if arr, ok := kv.Value.([]interface{}); ok {
			for _, v := range arr {
				if str, ok := v.(string); ok {
					header.Add(key, str)
				} else if i, ok := v.(int); ok {
					header.Add(key, strconv.Itoa(i))
				}
			}
			continue
		}

		return fmt.Errorf("UnmarshalYAML: headers: it was expected string or []string to field '%s' but %T", key, kv.Value)
	}

	return nil
}
