package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
)

// downloads all the current recipes and count the ones that are only filter expressions

const (
	baseUrl = "https://normandy.cdn.mozilla.net/api/v3/recipe/"
)

type stats struct {
	count int
	fo    int
}

var (
	m        sync.Mutex
	statList map[string]*stats
	todo     chan string
)

func init() {
	statList = make(map[string]*stats)
	todo = make(chan string, 10)
}

func get(url string) ([]byte, error) {
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

	return buf.Bytes(), nil
}

func process(body []byte) error {
	m.Lock()
	defer m.Unlock()

	jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

		// only do 2019 records
		created, err := jsonparser.GetString(value, "latest_revision", "updated")
		if err != nil {
			fmt.Println("Iteration error", err.Error())
		} else if !strings.Contains(created, "2019") {
			return
		}

		key := created[0:7]
		stat, ok := statList[key]
		if !ok { // create it if it doesn't exist
			stat = &stats{}
			statList[key] = stat
		}

		// an *exclusively* filter_object recipe should:
		//   - extra_filter_expression should be ""
		//   - filter_objects should have 1 or more elements

		extra, _ := jsonparser.GetString(value, "latest_revision", "extra_filter_expression")
		fobjects, _, _, _ := jsonparser.Get(value, "latest_revision", "filter_object")

		stat.count++
		if len(extra) == 0 && len(fobjects) > 3 { // fobjects = []byte("[]") when empty
			stat.fo++
		}

	}, "results")

	return nil
}

func main() {
	// fetch the base url to determine records and total count
	b, err := get(baseUrl)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	c, _ := jsonparser.GetInt(b, "count")

	process(b)

	pages := c/25 + 1

	// great N workers to load and process
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case url := <-todo:
					if url == "" { // receive on a closed channel
						return
					}

					if url == "done" {
						close(todo)
						return
					}

					body, err := get(url)
					if err != nil {
						fmt.Println(url, err)
						return
					} else {
						fmt.Println("Processing: ", url)
						process(body)
					}
				}
			}
		}()
	}

	for i := 2; i <= int(pages); i++ {
		todo <- fmt.Sprintf("%s?page=%d", baseUrl, i)
	}

	todo <- "done"

	wg.Wait()

	// ugly way to print things in a sorted month order
	for i := 1; i <= 12; i++ {
		key := fmt.Sprintf("2019-%02d", i)
		if stat, ok := statList[key]; ok {
			fmt.Printf("%s: Total: %d, FO: %d, PCT: %0.2f%%\n", key, stat.count, stat.fo, float64(stat.fo)/float64(stat.count)*100)
		}
	}
}
