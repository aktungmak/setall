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
	"net"
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

func MakeSetParamRequestTo(ipa, xpath, value string) (msg string, err error) {
	// generate an XML string of the RPC
	mc := NewMethodCall("setBoxParameters", xpath, value)
	ox, err := xml.MarshalIndent(mc, "", "  ")
	if err != nil {
		return "", err
	}

	// send the POST request and check for HTTP errors
	tgturi := fmt.Sprintf("http://%s/xmlrpc.cgi", ipa)
	body := bytes.NewBuffer(ox)
	r, err := http.Post(tgturi, "text/xml", body)
	if err != nil {
		return "", err
	}

	// parse the response and check for parse errors
	root, err := xmlpath.Parse(r.Body)
	if err != nil {
		return "", err
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
	return msg, err
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
		txt := scanner.Text()
		isip := net.ParseIP(txt)
		if isip != nil {
			lines = append(lines, txt)
		}
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
		if len(pair) == 2 {
			params = append(params, pair)
		}
	}

	return params, scanner.Err()
}

func main() {
	var ipafile, xpathfile string
	var debugmode bool
	flag.StringVar(&ipafile, "a", "REQUIRED",
		"file containing list of unit IP addresses, one per line")
	flag.StringVar(&xpathfile, "c", "REQUIRED",
		"file containing list of xpath, value command pairs, one pair per line")
	flag.BoolVar(&debugmode, "d", false,
		"enables debug mode, for extra detail of RPCs being sent")
	flag.Parse()

	if xpathfile == "REQUIRED" || ipafile == "REQUIRED" {
		log.Fatal("Incorrect usage, use --help for more information.\n\n")
	}

	// read the configuration files provided
	ipas, err := ParseAddrFile(ipafile)
	xpaths, err := ParseXpathFile(xpathfile)
	if err != nil {
		log.Fatal(err)
	}

	// now, do the actual work of setting values!
	errcnt := 0
	for i, xp := range xpaths {
		for j, ipa := range ipas {
			res, err := MakeSetParamRequestTo(ipa, xp[0], xp[1])
			if debugmode {
				fmt.Printf("\n>>>%s\n", res)
			}
			if err != nil {
				errcnt += 1
				fmt.Println("")
				fmt.Println(xp, ipa, "ERROR", err)
			} else {
				fmt.Printf("\rSetting param %d of %d on unit %d of %d...    ",
					i+1, len(xpaths), j+1, len(ipas))

			}
		}
	}
	fmt.Println("")
	fmt.Printf("Completed with %d errors.\n\n", errcnt)
}
