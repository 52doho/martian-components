// Package requestbody registers a request modifier for modify request bodies
package requestbody

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/devopsfaith/flatmap/tree"
	"github.com/google/martian/parse"
)

func init() {
	parse.Register("modify_request_body", FromJSON_modify_request_body)
	parse.Register("copy_urlquery_to_body", FromJSON_copy_urlquery_to_body)
}

func FromJSON_modify_request_body(b []byte) (*parse.Result, error) {
	var config []config
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, err
	}
	ops := []flatmapOp{}
	for _, c := range config {
		op := flatmapOp{}
		op.Type = c.Type
		op.Args = make([][]string, len(c.Args))
		for k, arg := range c.Args {
			op.Args[k] = strings.Split(arg, ".")
		}
		ops = append(ops, op)
	}

	msg := RequestBodyModifier{
		Ops: ops,
	}

	return parse.NewResult(&msg, []parse.ModifierType{parse.Request})
}

type RequestBodyModifier struct {
	Ops []flatmapOp
}

type config struct {
	Type string   `json:"type"`
	Args []string `json:"args"`
}

type flatmapOp struct {
	Type string     `json:"type"`
	Args [][]string `json:"args"`
}

func (m *RequestBodyModifier) ModifyRequest(req *http.Request) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}

	var bodyJson map[string]interface{}
	if err := json.Unmarshal(body, &bodyJson); err != nil {
		log.Printf("json.Unmarshal err: %v", err)
		return nil
	}

	flatten, err := tree.New(bodyJson)
	if err != nil {
		return err
	}
	for _, op := range m.Ops {
		switch op.Type {
		case "move":
			flatten.Move(op.Args[0], op.Args[1])
		case "append":
			flatten.Append(op.Args[0], op.Args[1])
		case "del":
			for _, k := range op.Args {
				flatten.Del(k)
			}
		case "add":
			flatten.Add(op.Args[0], op.Args[1][0])
		default:
		}
	}

	var bodyModified map[string]interface{}
	bodyModified, _ = flatten.Get([]string{}).(map[string]interface{})
	data, err := json.Marshal(bodyModified)
	if err != nil {
		return err
	}

	req.ContentLength = int64(len(data))
	req.Body = ioutil.NopCloser(bytes.NewReader(data))

	return nil
}

// copy_urlquery_to_body
type CopyUrlQueryToBodyModifier struct {
	Keys []string
}

func FromJSON_copy_urlquery_to_body(b []byte) (*parse.Result, error) {
	var keys []string
	if err := json.Unmarshal(b, &keys); err != nil {
		return nil, err
	}
	msg := CopyUrlQueryToBodyModifier{
		Keys: keys,
	}
	return parse.NewResult(&msg, []parse.ModifierType{parse.Request})
}

func (m *CopyUrlQueryToBodyModifier) ModifyRequest(req *http.Request) error {
	body, err := ioutil.ReadAll(req.Body)

	var bodyJson map[string]interface{}
	if err := json.Unmarshal(body, &bodyJson); err != nil {
		log.Printf("CopyUrlQueryToBodyModifier json.Unmarshal err: %v", err)
		bodyJson = make(map[string]interface{})
	}
	// copy url query to body
	query := req.URL.Query()
	for _, k := range m.Keys {
		bodyJson[k] = query[k]
	}

	data, err := json.Marshal(bodyJson)
	if err != nil {
		return err
	}

	req.ContentLength = int64(len(data))
	req.Body = ioutil.NopCloser(bytes.NewReader(data))

	return nil
}
