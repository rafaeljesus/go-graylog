package graylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func msDecode(input, output interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil, Result: output, TagName: "json",
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func writeOr500Error(w http.ResponseWriter, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		write500Error(w)
	} else {
		w.Write(b)
	}
}

func write500Error(w http.ResponseWriter) {
	w.WriteHeader(500)
	w.Write([]byte(`{"message":"500 Internal Server Error"}`))
}

func writeApiError(w http.ResponseWriter, code int, msg string, args ...interface{}) {
	w.WriteHeader(code)
	w.Write([]byte(fmt.Sprintf(
		`{"type": "ApiError", "message": "%s"}`, fmt.Sprintf(msg, args...))))
}

func getServerAndClient() (*MockServer, *Client, error) {
	server, err := NewMockServer("")
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to Get Mock Server")
	}
	client, err := NewClient(server.endpoint, "admin", "password")
	if err != nil {
		server.Close()
		return nil, nil, errors.Wrap(err, "Failed to NewClient")
	}
	server.Start()
	return server, client, nil
}

func (client *Client) callReq(
	ctx context.Context, method, endpoint string,
	body []byte, isReadBody bool,
) (*ErrorInfo, error) {
	var reqBody io.Reader = nil
	if body != nil {
		reqBody = bytes.NewBuffer(body)
	}
	req, err := http.NewRequest(method, endpoint, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to http.NewRequest")
	}
	ei := &ErrorInfo{Request: req}
	req.SetBasicAuth(client.GetName(), client.GetPassword())
	req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return ei, errors.Wrap(
			err, fmt.Sprintf("Failed to call Graylog API: %s %s", method, endpoint))
	}
	ei.Response = resp

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return ei, errors.Wrap(err, "Failed to read response body")
		}
		ei.ResponseBody = b
		if err := json.Unmarshal(b, ei); err != nil {
			return ei, errors.Wrap(
				err, fmt.Sprintf(
					"Failed to parse response body as ErrorInfo: %s", string(b)))
		}
		return ei, errors.New(ei.Message)
	}

	if isReadBody {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return ei, errors.Wrap(err, "Failed to read response body")
		}
		ei.ResponseBody = b
	}
	return ei, nil
}

func addToStringArray(arr []string, val string) []string {
	for _, v := range arr {
		if v == val {
			return arr
		}
	}
	return append(arr, val)
}

func removeFromStringArray(arr []string, val string) []string {
	ret := []string{}
	for _, v := range arr {
		if v != val {
			ret = append(ret, v)
		}
	}
	return ret
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func validateRequestBody(
	b []byte, requiredFields, allowedFields, acceptedFields []string,
) (
	int, string, map[string]interface{},
) {
	var a interface{}
	if err := json.Unmarshal(b, &a); err != nil {
		return 400, fmt.Sprintf(
			"Failed to parse the request body as JSON: %s (%s)", string(b), err), nil
	}
	body, ok := a.(map[string]interface{})
	if !ok {
		return 400, fmt.Sprintf(
			"Failed to parse the request body as a JSON object : %s", string(b)), nil
	}
	rqf := makeHash(requiredFields)
	for k, _ := range rqf {
		if _, ok := body[k]; !ok {
			return 400, fmt.Sprintf(
				"In the request body the field %s is required", k), body
		}
	}
	alf := makeHash(allowedFields)
	if len(alf) != 0 {
		for k, _ := range rqf {
			alf[k] = nil
		}
		arr := make([]string, len(alf))
		i := 0
		for k, _ := range alf {
			arr[i] = k
			i++
		}
		for k, _ := range body {
			if _, ok := alf[k]; !ok {
				return 400, fmt.Sprintf(
					"In the request body an invalid field is found: %s\nThe allowed fields: %s, request body: %s",
					k, strings.Join(arr, ", "), string(b)), body
			}
		}
	}
	acf := makeHash(acceptedFields)
	if len(acf) != 0 {
		for k, _ := range rqf {
			acf[k] = nil
		}
		for k, _ := range alf {
			acf[k] = nil
		}
		for k, _ := range body {
			if _, ok := acf[k]; !ok {
				delete(body, k)
			}
		}
	}
	return 200, "", body
}

func makeHash(arr []string) map[string]interface{} {
	if arr == nil {
		return map[string]interface{}{}
	}
	h := map[string]interface{}{}
	for _, k := range arr {
		h[k] = nil
	}
	return h
}
