package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// HTTPTransport sends and receives JSON messages to a HTTP server.
type HTTPTransport struct {
	Server string
	client *http.Client
}

// ApiError is an error message returned by the API server on errors.
type ApiError struct {
	Status      int    `json:"status"`
	ErrorName   string `json:"error"`
	Description string `json:"description"`
	Message     string `json:"message"`
	Stacktrace  string `json:"stacktrace"`
}

// NewHTTPTransport returns a new HTTPTransport.
func NewHTTPTransport(serverURL string) *HTTPTransport {
	url := serverURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return &HTTPTransport{
		Server: url,
		client: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

func (transport *HTTPTransport) request(url string, method string, result interface{}, object interface{}) *Error {
	if method != http.MethodPost && method != http.MethodGet {
		panic("Unsupported HTTP method " + method)
	}
	if method == http.MethodGet && object != nil {
		panic("Cannot GET and also post an object")
	}

	var isstr bool
	var reader io.Reader
	if object != nil {
		var objstr string
		if objstr, isstr = object.(string); isstr {
			reader = bytes.NewBuffer([]byte(objstr))
		} else {
			marshaled, err := json.Marshal(object)
			if err != nil {
				return &Error{error: err, ErrorCode: ErrorSerialization}
				//return &TransportError{Err: err.Error()}
			}
			//fmt.Printf("POST: %s\n", string(marshaled))
			reader = bytes.NewBuffer(marshaled)
		}
	}

	req, err := http.NewRequest(method, transport.Server+url, reader)
	if err != nil {
		return &Error{error: err, ErrorCode: ErrorTransport}
	}

	req.Header.Set("User-Agent", "irmago")
	if object != nil {
		if isstr {
			req.Header.Set("Content-Type", "text/plain; charset=UTF-8")
		} else {
			req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		}
	}

	res, err := transport.client.Do(req)
	if err != nil {
		return &Error{error: err, ErrorCode: ErrorTransport}
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return &Error{error: err, Status: res.StatusCode}
	}
	if res.StatusCode != 200 {
		apierr := &ApiError{}
		json.Unmarshal(body, apierr)
		if apierr.ErrorName == "" { // Not an ApiErrorMessage
			return &Error{ErrorCode: ErrorTransport, Status: res.StatusCode}
		}
		//fmt.Printf("ERROR: %+v\n", apierr)
		return &Error{ErrorCode: ErrorTransport, Status: res.StatusCode, ApiError: apierr}
	}

	//fmt.Printf("RESPONSE: %s\n", string(body))
	err = json.Unmarshal(body, result)
	if err != nil {
		return &Error{error: err, Status: res.StatusCode}
	}

	return nil
}

// Post sends the object to the server and parses its response into result.
func (transport *HTTPTransport) Post(url string, result interface{}, object interface{}) *Error {
	return transport.request(url, http.MethodPost, result, object)
}

// Get performs a GET request and parses the server's response into result.
func (transport *HTTPTransport) Get(url string, result interface{}) *Error {
	return transport.request(url, http.MethodGet, result, nil)
}

// Delete performs a DELETE.
func (transport *HTTPTransport) Delete(url string) {
	// TODO
}
