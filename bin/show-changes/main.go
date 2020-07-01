package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/mostlygeek/normandy-tools/tools"
)

// This outputs a list of information with:
// - first revision, last revision, days in between
// - is the recipe live?
// - recipe id, type and slug
//
// this information is useful for getting a high level view of what's currently
// live in production.  Also useful to see what has ended, when it ended, etc.
//
type ChangeRevision struct {
	Enabled bool
	Time    string
}
type Record struct {
	Id        int
	Action    string
	Slug      string
	Revisions []ChangeRevision
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
	r.Lock()
	defer r.Unlock()
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
				body, err := tools.Get(url)
				if err != nil {
					fmt.Println("Error fetching revisions: ", url)
				} else {
					record := revisions.Get(id)

					jsonparser.ArrayEach(body, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
						created, _ := jsonparser.GetString(value, "date_created")
						enabled, _ := jsonparser.GetBoolean(value, "enabled")

						record.Revisions = append(record.Revisions, ChangeRevision{enabled, created})
					})

					// put it back (struct, not a pointer ...)
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

		body, err := tools.Get(next)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		next, err = jsonparser.GetString(body, "next")
		if err != nil {
			break
		}

		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			id64, err := jsonparser.GetInt(value, "id")
			if err != nil {
				fmt.Println("Error extracting id: ", err.Error())
				return
			}

			// convert it to an int
			id := int(id64)

			// only do 2019, 2020 records
			created, err := jsonparser.GetString(value, "latest_revision", "updated")
			if err != nil {
				fmt.Println("Iteration error", err.Error())
				return
			} else if !strings.Contains(created, "2019") && !strings.Contains(created, "2020") {
				return
			}

			action, _ := jsonparser.GetString(value, "latest_revision", "action", "name")
			slug, _ := jsonparser.GetString(value, "latest_revision", "arguments", "slug")

			// Filter out unwanted types
			switch action {
			case "console-log":
				return
			default:
				// created the record into the data
				record := Record{Id: id, Action: action, Slug: slug}
				revisions.Set(id, record)
				revisionTodo <- id
			}

		}, "results")
	}

	revisionTodo <- -1

	// wait for all the revision pulling to finish
	wg.Wait()

	// Process all the data
	for _, rec := range revisions.Data() {
		//fmt.Println(id, rec.Action, rec.Slug)

		// find the earliest / oldest dates
		lastEnabled := false
		var tsFirst, tsLast string
		var earliest, latest int64

		// revisions seem to be in sorted order but lets not assume things
		for _, revision := range rec.Revisions {
			ts := tools.RFC3339ToUnix(revision.Time)
			if earliest == 0 || earliest > ts {
				earliest = ts
				tsFirst = revision.Time
			}

			if latest == 0 || latest < ts {
				latest = ts
				lastEnabled = revision.Enabled
				tsLast = revision.Time
			}
		}

		if rec.Slug == "" {
			rec.Slug = "--"
		}

		enableTime := latest - earliest
		if lastEnabled == true {
			enableTime = time.Now().Unix() - earliest
		}
		// dump the data
		fmt.Println(rec.Id, rec.Action, rec.Slug, lastEnabled, tsFirst[0:10], tsLast[0:10], (enableTime / 86400), len(rec.Revisions))

	}
}
