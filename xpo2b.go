// provides methods for configuring avps/vpcs through
// the XMLRPC interface
package xpo2b

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"gopkg.in/xmlpath.v2"
	"io/ioutil"
	"net/http"
	"strings"
)

//struct to describe an XMLRPC payload
type MethodCall struct {
	XMLName    xml.Name      `xml:"methodCall"`
	MethodName string        `xml:"methodName"`
	Params     []StringParam `xml:"params>param"`
}

// an individual string parameter
// this + XpoParam could be Param interfaces
type StringParam struct {
	// XMLName     xml.Name `xml:"param"`
	StringValue string `xml:"value>string"`
}

// describes a block of xml rather than just string
type XpoParam struct {
	XMLName  xml.Name `xml:"value"`
	XpoValue string   `xml:"param>value>xpo"`
}

// factory method to produce a new MethodCall struct
// with an arbitrary number of string params
func NewMethodCall(mname string, args ...string) MethodCall {
	var parms []StringParam
	var temp StringParam
	for _, arg := range args {
		temp = StringParam{StringValue: strings.TrimSpace(arg)}
		parms = append(parms, temp)
	}

	mc := MethodCall{
		MethodName: mname,
		Params:     parms,
	}
	return mc
}

func SendXMLRPCPayload(host string, payload MethodCall) (string, error) {
	ox, err := xml.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	// send the POST request and check for HTTP errors
	tgturi := fmt.Sprintf("http://%s/xmlrpc.cgi", host)
	body := bytes.NewBuffer(ox)
	r, err := http.Post(tgturi, "text/xml", body)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	ret, err := ioutil.ReadAll(r.Body)

	return string(ret), err

}

func ParseXMLRPCResponse(body string) error {
	// parse the response and check for parse errors
	root, err := xmlpath.Parse(body)
	if err != nil {
		return err
	}

	// examine the response and look for RPC errors
	faultpath := xmlpath.MustCompile("//fault/value/string")
	parampath := xmlpath.MustCompile("//param/value/string")

	if msg, found := parampath.String(root); found {
		// it maybe worked
		if len(msg) == 0 {
			// success!
			err = error(nil)
		} else {
			// there was a problem
			err = errors.New(msg)
		}
	} else if msg, found := faultpath.String(root); found {
		//it didn't work
		err = errors.New(msg)
	} else {
		err = errors.New("couldn't parse response!")
	}
	return err

}
