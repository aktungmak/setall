package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

type MethodCall struct {
	XMLName    xml.Name      `xml:"methodCall"`
	MethodName string        `xml:"methodName"`
	Params     []StringParam `xml:"params>param"`
}

// these could be Param interfaces
type StringParam struct {
	// XMLName     xml.Name `xml:"param"`
	StringValue string `xml:"value>string"`
}

// not used yet
type XpoParam struct {
	XMLName  xml.Name `xml:"value"`
	XpoValue string   `xml:"param>value>xpo"`
}

func NewMethodCall(mname, xpath, value string) MethodCall {
	xp := StringParam{StringValue: xpath}
	vl := StringParam{StringValue: value}
	parms := []StringParam{xp, vl}
	mc := MethodCall{
		MethodName: mname,
		Params:     parms,
	}
	return mc
}

func main() {
	fmt.Printf("this is setall\n")
	// mc := MethodCall{MethodName: "setParameters"}
	mc := NewMethodCall("setParameters", "/viper[1]/slotList/slot[1]/card", "12")
	ox, err := xml.MarshalIndent(mc, "", "  ")
	if err != nil {
		fmt.Printf("%s", err)
	}
	// fmt.Printf("%s%s", xml.Header, ox)
	body := bytes.NewBuffer(ox)
	r, _ := http.Post("http://httpbin.org/post", "text/xml", body)
	response, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(response))
	// just need to parse response and check for success or not
}
