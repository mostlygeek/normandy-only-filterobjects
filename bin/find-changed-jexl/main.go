package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
)

// creates a table with this data
// - id
// - type
// - num revisions
// - FO Used

type Record struct {
	Id                      int
	Action                  string
	LastRevision            string
	NumRevisions            int
	FilterObjectUsed        bool
	FilterExpressionChanges int
	LastFilterExpression    string
}

type RevisionData struct {
	sync.Mutex
	data map[int]Record
}

func NewRevisionData() *RevisionData {
	return &RevisionData{
		data: make(map[int]Record),
	}
}
func (r *RevisionData) Get(id int) Record {
	return r.data[id]
}
func (r *RevisionData) Set(id int, rec Record) {
	r.Lock()
	defer r.Unlock()
	r.data[id] = rec
}

func (r *RevisionData) Data() map[int]Record {
	return r.data
}

const (
	baseUrl = "https://normandy.cdn.mozilla.net/api/v3/recipe/"
)

var (
	revisionTodo     chan int
	recipesRevisions map[int]json.RawMessage
	revisions        = NewRevisionData()
)

func init() {
	revisionTodo = make(chan int, 10)
	recipesRevisions = make(map[int]json.RawMessage)
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

func tsToUnix(t string) int64 {
	if t == "" {
		return 0
	}
	tt, err := time.Parse(time.RFC3339, t)
	if err != nil {
		fmt.Println("Failed parsing", t, err.Error())
		return 0
	} else {
		return tt.Unix()
	}
}
func main() {

	// lots of workers to load and process data fast
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				id := <-revisionTodo
				if id == 0 { // channel was closed
					break
				}

				if id == -1 { // no more ids expected, close the channel
					close(revisionTodo)
					break
				}

				url := fmt.Sprintf("%s%d/history/", baseUrl, id)
				body, err := get(url)
				if err != nil {
					fmt.Println("Error fetching revisions: ", url)
				} else {
					record := Record{Id: id}

					jsonparser.ArrayEach(body, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
						record.Action, _ = jsonparser.GetString(value, "action", "name")
						record.NumRevisions++

						fo, _, _, err := jsonparser.Get(value, "filter_object")
						if err != nil {
							fmt.Println("FO Err:", err.Error())
						} else {
							if len(fo) > 2 { // more than an empty array []
								record.FilterObjectUsed = true
							}
						}

						// manage time stamps in record
						updated, _ := jsonparser.GetString(value, "updated")
						if tsToUnix(record.LastRevision) < tsToUnix(updated) {
							record.LastRevision = updated
						}

						fe, _ := jsonparser.GetString(value, "filter_expression")
						if record.LastFilterExpression != fe {
							record.LastFilterExpression = fe
							record.FilterExpressionChanges++
						}

					})

					revisions.Set(id, record)
				}
			}
		}()
	}

	next := baseUrl + "?ordering=-id"
	count := 0
	for {
		// useful when hacking to not load allllll the pages
		if count++; count > 100 {
			break
		}

		if next == "" {
			break
		}
		fmt.Println("Fetching:", next)

		body, err := get(next)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		next, err = jsonparser.GetString(body, "next")
		if err != nil {
			break
		}

		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			id, err := jsonparser.GetInt(value, "id")
			if err != nil {
				fmt.Println("Error extracting id: ", err.Error())
				return
			}

			// only do 2019, 2020 records
			created, err := jsonparser.GetString(value, "latest_revision", "updated")
			if err != nil {
				fmt.Println("Iteration error", err.Error())
				return
			} else if !strings.Contains(created, "2019") && !strings.Contains(created, "2020") {
				return
			}

			// Filter out all the heartbeat messages
			actionType, _ := jsonparser.GetString(value, "latest_revision", "action", "name")
			if actionType != "show-heartbeat" && actionType != "console-log" {
				revisionTodo <- int(id)
			}

		}, "results")
	}

	revisionTodo <- -1

	// wait for all the revision pulling to finish
	wg.Wait()

	// Process all the data
	for id, rec := range revisions.Data() {
		fmt.Println(id, rec.NumRevisions, rec.Action, rec.LastRevision, rec.FilterObjectUsed, rec.FilterExpressionChanges)
	}

}
