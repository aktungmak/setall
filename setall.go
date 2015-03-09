package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"gopkg.in/xmlpath.v2"
	"log"
	"net/http"
	"os"
	"strings"
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
	xp := StringParam{StringValue: strings.TrimSpace(xpath)}
	vl := StringParam{StringValue: strings.TrimSpace(value)}
	parms := []StringParam{xp, vl}
	mc := MethodCall{
		MethodName: mname,
		Params:     parms,
	}
	return mc
}

func MakeSetParamRequestTo(ipa, xpath, value string) (string, error) {
	mc := NewMethodCall("setParameters", xpath, value)
	ox, err := xml.MarshalIndent(mc, "", "  ")
	if err != nil {
		return "", err
	}
	// fmt.Printf("%s%s", xml.Header, ox)
	tgturi := fmt.Sprintf("http://%s/xmlrpc.cgi", ipa)
	body := bytes.NewBuffer(ox)
	r, err := http.Post(tgturi, "text/xml", body)
	if err != nil {
		var reason string
		if err.Timeout() {
			reason = "timed out"
		} else {
			reason = "request failed"
		}
		return "", errors.New(reason)
	}

	root, err := xmlpath.Parse(r.Body)
	if err != nil {
		return "", err
	}
	resultpath := xmlpath.MustCompile("//string")
	if value, ok := resultpath.String(root); ok {
		if len(value) == 0 {
			return value, nil
		} else {
			return "", errors.New(value)
		}
	} else {
		return "", errors.New("couldn't parse response!")
	}
}

func ParseAddrFile(fname string) ([]string, error) {
	var lines []string
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func ParseXpathFile(fname string) ([][]string, error) {
	var params [][]string
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		pair := strings.SplitN(scanner.Text(), ",", 2)
		params = append(params, pair)
	}

	return params, scanner.Err()
}

func main() {
	var ipafile, xpathfile string
	flag.StringVar(&ipafile, "a", "",
		"file containing list of unit IP addresses, one per line")
	flag.StringVar(&xpathfile, "x", "",
		"file containing list of xpath, value pairs, one pair per line")
	flag.Parse()

	ipas, err := ParseAddrFile(ipafile)
	xpaths, err := ParseXpathFile(xpathfile)
	if err != nil {
		log.Fatal(err)
	}

	for _, xp := range xpaths {
		for _, ipa := range ipas {
			res, err := MakeSetParamRequestTo(ipa, xp[0], xp[1])
			if err != nil {
				fmt.Println(xp, ipa, "ERROR", err)
			} else {
				fmt.Println(xp, ipa, "SET OK", res)
			}
		}
	}

}
