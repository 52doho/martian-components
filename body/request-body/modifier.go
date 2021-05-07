// Package requestbody registers a request modifier for modify request bodies
package krakend

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/devopsfaith/flatmap/tree"
	"github.com/google/martian/parse"
)

func init() {
	parse.Register("modify_request_body", FromJSON)
}

func FromJSON(b []byte) (*parse.Result, error) {
	var ops []flatmapOp
	if err := json.Unmarshal(b, &ops); err != nil {
		return nil, err
	}

	msg := RequestBodyModifier{
		Ops: ops,
	}

	return parse.NewResult(msg, []parse.ModifierType{parse.Request})
}

type RequestBodyModifier struct {
	Ops []flatmapOp
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
	req.Body.Close()

	var bodyJson map[string]interface{}
	if err := json.Unmarshal(body, bodyJson); err != nil {
		return err
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
