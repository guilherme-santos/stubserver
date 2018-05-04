package stubserver

import (
	"fmt"
	"net/http"
	"net/url"
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

func (c *ConfigRequest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Orig ConfigRequest
	hack := struct {
		*Orig
		Headers  yaml.MapSlice
		Response interface{} // can be string or ConfigResponse
	}{
		Orig: (*Orig)(c),
	}

	err := unmarshal(&hack)
	if err != nil {
		return err
	}

	_, err = url.ParseRequestURI(c.URL)
	if err != nil {
		return fmt.Errorf("url is not valid: %s", err)
	}
	if c.Method == "" {
		c.Method = "GET"
	}

	c.URL = strings.TrimSpace(c.URL)

	return mapSliceToHeader(hack.Headers, c.Headers)
}

func (r *ConfigResponse) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var data string
	err := unmarshal(&data)
	if err == nil {
		r.Headers = http.Header{}
		r.Data = data
		return nil
	}

	type Orig ConfigResponse
	hack := struct {
		*Orig
		Headers yaml.MapSlice
	}{
		Orig: (*Orig)(r),
	}

	err = unmarshal(&hack)
	if err != nil {
		return err
	}

	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}

	return mapSliceToHeader(hack.Headers, r.Headers)
}

func mapSliceToHeader(mapSlice yaml.MapSlice, header http.Header) error {
	if header == nil {
		header = http.Header{}
	}

	for _, kv := range mapSlice {
		key := kv.Key.(string)

		if str, ok := kv.Value.(string); ok {
			header.Set(key, str)
			continue
		}

		if arr, ok := kv.Value.([]interface{}); ok {
			for _, v := range arr {
				v := v.(string)
				header.Add(key, v)
			}
			continue
		}

		return fmt.Errorf("UnmarshalYAML: headers: it was expected string or []string to field '%s' but %T", key, kv.Value)
	}

	return nil
}
