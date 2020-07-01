package tools

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

const (
	cachedir = "/tmp/normandy-tools-cache/"
)

func init() {

	// just make sure its there
	if err := os.Mkdir(cachedir, 0755); err != nil && !os.IsExist(err) {
		fmt.Println("MKDIR err: ", err.Error())
	}

}

func cachefilename(url string) string {
	h := md5.New()
	io.WriteString(h, url)
	return cachedir + hex.EncodeToString(h.Sum(nil))
}

func cacheget(url string) ([]byte, bool) {
	data, err := ioutil.ReadFile(cachefilename(url))
	return data, (err == nil)
}

func cachewrite(url string, data []byte) error {
	return ioutil.WriteFile(cachefilename(url), data, 0644)
}

func Cachedir() string { return cachedir }
func Get(url string) ([]byte, error) {

	// attempt to get from cache
	if body, ok := cacheget(url); ok {
		return body, nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

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

	body := buf.Bytes()

	if err := cachewrite(url, body); err != nil {
		// whatever, good enough for the cli apps :D
		fmt.Println("Unable to cache body", err.Error())
	}

	return body, nil
}
