package schemaregistry

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const testHost = "testhost:1337"
const testURL = "http://" + testHost

type D func(req *http.Request) (*http.Response, error)

func (d D) Do(req *http.Request) (*http.Response, error) {
	return d(req)
}

// verifies the http.Request, creates an http.Response
func dummyHTTPHandler(t *testing.T, method, path string, status int, reqBody, respBody interface{}, contentType string) D {
	d := D(func(req *http.Request) (*http.Response, error) {
		if method != "" && req.Method != method {
			t.Errorf("method is wrong, expected `%s`, got `%s`", method, req.Method)
		}
		if req.URL.Host != testHost {
			t.Errorf("expected host `%s`, got `%s`", testHost, req.URL.Host)
		}
		if path != "" && req.URL.Path != path {
			t.Errorf("path is wrong, expected `%s`, got `%s`", path, req.URL.Path)
		}
		if reqBody != nil {
			expbs, err := json.Marshal(reqBody)
			if err != nil {
				t.Error(err)
			}
			bs, err := ioutil.ReadAll(req.Body)
			mustEqual(t, strings.Trim(string(bs), "\r\n"), strings.Trim(string(expbs), "\r\n"))
		}
		var resp http.Response
		resp.Header = http.Header{contentTypeHeaderKey: []string{contentType}}
		resp.StatusCode = status
		if respBody != nil {
			bs, err := json.Marshal(respBody)
			if err != nil {
				t.Error(err)
			}
			resp.Body = ioutil.NopCloser(bytes.NewReader(bs))
		}
		return &resp, nil
	})
	return d
}

func httpSuccess(t *testing.T, method, path string, reqBody, respBody interface{}, contentType string) *Client {
	return &Client{testURL, dummyHTTPHandler(t, method, path, 200, reqBody, respBody, contentType)}
}

func httpError(t *testing.T, status, errCode int, errMsg, contentType string) *Client {
	return &Client{testURL, dummyHTTPHandler(t, "", "", status, nil, ResourceError{ErrorCode: errCode, Message: errMsg}, contentType)}
}

func mustEqual(t *testing.T, actual, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected `%#v`, got `%#v`", expected, actual)
	}
}

func TestSubjects(t *testing.T) {
	subsIn := []string{"rollulus", "hello-subject"}
	c := httpSuccess(t, "GET", "/subjects", nil, subsIn, contentTypeSchemaJSON)
	subs, err := c.Subjects()
	if err != nil {
		t.Error()
	}
	mustEqual(t, subs, subsIn)
}

func TestVersions(t *testing.T) {
	versIn := []int{1, 2, 3}
	c := httpSuccess(t, "GET", "/subjects/mysubject/versions", nil, versIn, contentTypeSchemaJSON)
	vers, err := c.Versions("mysubject")
	if err != nil {
		t.Error()
	}
	mustEqual(t, vers, versIn)
}

func TestIsRegistered_yes(t *testing.T) {
	s := `{"x":"y"}`
	ss := schemaOnlyJSON{s}
	sIn := Schema{s, "mysubject", 4, 7}
	c := httpSuccess(t, "POST", "/subjects/mysubject", ss, sIn, contentTypeSchemaJSON)
	isreg, sOut, err := c.IsRegistered("mysubject", s)
	if err != nil {
		t.Error()
	}
	if !isreg {
		t.Error()
	}
	mustEqual(t, sOut, sIn)
}

func TestIsRegistered_not(t *testing.T) {
	c := httpError(t, 404, schemaNotFoundCode, "too bad", contentTypeSchemaJSON)
	isreg, _, err := c.IsRegistered("mysubject", "{}")
	if err != nil {
		t.Fatal(err)
	}
	if isreg {
		t.Fatalf("is registered: %v", err)
	}
}

func TestIsRegisteredWrongHeader(t *testing.T) {
	c := httpError(t, 500, 50103, " wrong content type", contentTypeJSON)
	isreg, _, err := c.IsRegistered("mysubject", "{}")
	mustEqual(t, isreg, false)
	mustEqual(t, err.Error(), "client: (: ) failed with error code 50103 wrong content type")
}

func TestVersionCheck(t *testing.T) {
	err := checkSchemaVersionID(29)
	if err != nil {
		t.Fatal(err)
	}
}
