// Logspout adapter to push events to Azure Operations Management Suite.
//
// Use as
// oms://<workspace-id>.ods.opinsights.azure.com?sharedKey=<urlencoded key>'
package oms

// The MIT License (MIT)
// =====================
//
// Copyright © 2017 Kungliga Tekniska högskolan
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the “Software”), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewOmsAdapter, "oms")
}

func NewOmsAdapter(route *router.Route) (router.LogAdapter, error) {
	sharedKey := route.Options["sharedKey"]
	workspaceId := strings.Split(route.Address, ".")[0]
	uri := "https://" + workspaceId + ".ods.opinsights.azure.com/api/logs?api-version=2016-04-01"

	transport := &http.Transport{
	  Dial: (&net.Dialer{
	    Timeout: 5 * time.Second,
	  }).Dial,
	  TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: transport,
	}

	time.LoadLocation("Stockholm/Sweden")

	return &OmsAdapter{
		route: 			 route,
		uri:				 uri,
		workspaceId: workspaceId,
		sharedKey: 	 sharedKey,
		client: 		 client,
	}, nil
}

type OmsAdapter struct {
	route *router.Route
	uri string
	workspaceId string
	sharedKey string
	client  *http.Client
}

func (adapter *OmsAdapter) signature(stringToSign string) (signature string) {
	// Signature=Base64(HMAC-SHA256(UTF8(StringToSign)))
	key, _ := base64.StdEncoding.DecodeString(adapter.sharedKey)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	buffer := mac.Sum(nil)

	signature = base64.StdEncoding.EncodeToString(buffer)
	return signature
}

func (adapter *OmsAdapter) authorization(request *http.Request) (authorization string) {
	return "SharedKey " + adapter.workspaceId + ":" + adapter.signature(stringToSign(request))
}

func stringToSign(request *http.Request) (stringToSign string) {
	// POST\n1024\napplication/json\nx-ms-date:Mon, 04 Apr 2016 08:00:00 GMT\n/api/logs
	stringToSign =
		request.Method + "\n" +
		strconv.FormatInt(request.ContentLength, 10) + "\n" +
 		request.Header.Get("Content-Type") + "\n" +
		"x-ms-date:" + request.Header.Get("x-ms-date") + "\n" +
		"/api/logs"
	return stringToSign
}

func (adapter *OmsAdapter) makeRequest(logType string, body []byte) (request *http.Request) {
	request, err := http.NewRequest("POST", adapter.uri, bytes.NewReader(body))
	if err != nil {
		log.Println("logspout-oms: error:", err)
		return nil
	}

	request.Header.Add("Log-Type", logType)
	request.Header.Add("Content-Type", "application/json")
	// OMS really requires 'GMT' rather than non-compliant time.RFC1123
	request.Header.Add("x-ms-date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	request.Header.Add("authorization", adapter.authorization(request))
	return request
}

func level(source string) (level int, levelStr string) {
	if (source == "stdout") {
		return 30, "INFO"
	} else {
		return 50, "ERROR"
	}
}

func (adapter *OmsAdapter) sendDefault(message *router.Message) {
	level, levelStr := level(message.Source)
	msg := BunyanMessage {
		V: 			  0,
		Level:	  level,
		LevelStr: levelStr,
		Name:			message.Container.Name,
		Hostname: message.Container.Config.Hostname,
		Pid:		  message.Container.ID,
		Time:     time.Now().Format("2006-01-02T15:04:05.999Z"),
		Msg: 		  message.Data,
		Src:		  message.Container.Config.Image,
		DockerInfo:   dockerinfo(message),
	}

	if body, err := json.Marshal(msg); err == nil {
		adapter.send("Bunyan", body)
	} else {
  	log.Println("logstash: could not marshal JSON:", err)
	}
}


func (adapter *OmsAdapter) sendJson(data map[string]interface{}) {
	if body, err := json.Marshal(data); err == nil {
		if data["Type"] != nil {
			adapter.send(data["Type"].(string), body)
		} else {
			adapter.send("Bunyan", body)
		}
	} else {
  	log.Println("logstash: could not marshal JSON:", err)
	}
}


func dockerinfo(message *router.Message) (dockerinfo DockerInfo) {
	return DockerInfo {
		Name:     message.Container.Name,
		ID:       message.Container.ID,
		Image:    message.Container.Config.Image,
		Hostname: message.Container.Config.Hostname,
		Labels:   message.Container.Config.Labels,
	}
}


func (adapter *OmsAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		var data map[string]interface{}

		if err := json.Unmarshal([]byte(message.Data), &data); err != nil {
			// Generate JSON from message and send.
			adapter.sendDefault(message);
		} else {
			// Add docker meta data and send json as is.
			data["dockerinfo"] = dockerinfo(message)
			adapter.sendJson(data);
		}
	}
}


func (adapter *OmsAdapter) send(logType string, body []byte) {
	for attempt := 1; attempt <= 10; attempt++ {
		request := adapter.makeRequest(logType, body)
		response, err := adapter.client.Do(request)

		if err != nil {
			log.Println("logspout-oms:", err)
			if response != nil {
				io.Copy(ioutil.Discard, response.Body)
				response.Body.Close()
			}
		} else if ! ((response.StatusCode == 200) || (response.StatusCode == 202)) {
			log.Println("logspout-oms: status:", response.Status)
			buf := new(bytes.Buffer)
			response.Write(buf)
			log.Println("logspout-oms: response:", buf.String())
			response.Body.Close()
		} else {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
			return
		}
		log.Println("logspout-oms: back-off in seconds before retry:", attempt)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	log.Println("logspout-oms: unable to send message, exiting.")
	os.Exit(1)
}


type DockerInfo struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Image    string `json:"image"`
	Hostname string `json:"hostname"`
	Labels  map[string]string `json:"labels"`
}

type BunyanMessage struct {
	V				 int      `json:"v"`
	Level		 int      `json:"level"`
	LevelStr string   `json:"levelStr"`
	Name	   string   `json:"name"`
	Hostname string   `json:"hostname"`
	Pid      string   `json:"pid"`
	Time     string   `json:"time"`
	Msg      string   `json:"msg"`
	Src      string   `json:"src"`
	DockerInfo DockerInfo `json:"dockerinfo"`
}
