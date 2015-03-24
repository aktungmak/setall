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

func MakeSetModeMLRequestTo(ipa string, index uint, state string) (err error) {
	// generate an XML string of the RPC

	var xpath string
	if index == 0 {
		xpath = "/viper/outputList/output[2]/transportStreamList/transportStream"
	} else {
		xpath = fmt.Sprintf("/viper/outputList/output[2]/transportStreamList/transportStream[%d]", index)
	}
	mc := NewMethodCall("setModeMediaLevel", xpath, state, "")
	ox, err := xml.MarshalIndent(mc, "", "  ")
	if err != nil {
		return err
	}

	// send the POST request and check for HTTP errors
	tgturi := fmt.Sprintf("http://%s/xmlrpc.cgi", ipa)
	body := bytes.NewBuffer(ox)
	r, err := http.Post(tgturi, "text/xml", body)
	if err != nil {
		return err
	}

	// parse the response and check for parse errors
	root, err := xmlpath.Parse(r.Body)
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

func main() {
	var ipafile string
	var state, tsindex uint

	// parse the cmdline args
	flag.StringVar(&ipafile, "a", "REQUIRED",
		"file containing list of unit IP addresses, one per line")
	flag.UintVar(&tsindex, "i", 0,
		"The 1-based index of the IP-out TS to enable. 0 will set all TS on that chassis.")
	flag.UintVar(&state, "s", 1,
		"The state to set. 0 = Offline, 1 = Online.")
	flag.Parse()
	if ipafile == "REQUIRED" {
		log.Fatal("Incorrect usage, use --help for more information.\n\n")
	}

	// read the configuration file provided
	ipas, err := ParseAddrFile(ipafile)
	if err != nil {
		log.Fatal(err)
	}

	// turn state arg into actual string
	var statestr string
	switch state {
	case 0:
		statestr = "Offline"
	case 1:
		statestr = "Online"
	default:
		log.Fatal("State must be either 0 (Offline) or 1 (Online)\n\n")
	}

	// now actually send the requests to each device
	errcnt := 0
	for j, ipa := range ipas {
		err := MakeSetModeMLRequestTo(ipa, tsindex, statestr)
		if err != nil {
			errcnt += 1
			fmt.Println("")
			fmt.Println(ipa, "ERROR", err)
		} else {
			fmt.Printf("\rSetting TS %s on unit %d of %d...    ",
				statestr, j+1, len(ipas))
		}
	}
	fmt.Println("")
	fmt.Printf("Completed with %d errors.\n\n", errcnt)
}
