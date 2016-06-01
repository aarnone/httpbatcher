// +build integration

package main

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var dockerHostname string

func init() {
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost == "" {
		dockerHostname = "localhost"
	} else {
		dockerHostURL, err := url.Parse(dockerHost)
		if err != nil {
			panic(err)
		}

		dockerHostname = dockerHostURL.Host
	}
}

func TestMixedCalls(t *testing.T) {
	requestBody := []byte(`POST /batch HTTP/1.1
Host: ` + dockerHostname + `:8080
Content-Length: 469
Content-Type: multipart/mixed; boundary="===============7330845974216740156=="


--===============7330845974216740156==
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: <b29c5de2-0db4-490b-b421-6a51b598bd22+1>

GET /some/thing HTTP/1.1
Host: serverA:8080
Accept: */*


--===============7330845974216740156==
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: <b29c5de2-0db4-490b-b421-6a51b598bd22+2>

POST http://serverB:8080/weeee HTTP/1.1
Accept: */*


--===============7330845974216740156==--`)

	t.Log(string(requestBody))

	conn, err := net.DialTimeout("tcp", dockerHostname+":8080", 5*time.Second)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	_, err = conn.Write(requestBody)
	if err != nil {
		panic(err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	logResponse(t, resp)
}

func logResponse(t *testing.T, resp *http.Response) {
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Log(err)
		return
	}

	t.Log(string(dump))
}
