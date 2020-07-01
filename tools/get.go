package tools

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/pkg/errors"
)

var (
	hclient  http.Client
	cachedir string
)

func init() {

	cachedir = "/tmp/normandy-tools-cache/"

	// just make sure its there
	if err := os.Mkdir(cachedir, 0755); err != nil && !os.IsExist(err) {
		fmt.Println("MKDIR err: ", err.Error())
	}

	cache := diskcache.New(cachedir)
	tp := httpcache.NewTransport(cache)

	hclient = http.Client{
		Transport: tp,
	}

}

func Cachedir() string { return cachedir }
func Get(url string) ([]byte, error) {
	resp, err := hclient.Get(url)
	if err != nil {
		return nil, err
	}

	//fromCache := resp.Header.Get("X-From-Cache")
	//fmt.Println(url, "cached:", fromCache)

	if resp.Body == nil {
		return nil, errors.New("Empty Body")
	}

	if resp.StatusCode != 200 {
		return nil, errors.Errorf("Response Code is %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
