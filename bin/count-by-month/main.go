package main

import (
	"fmt"
	"sync"

	"github.com/buger/jsonparser"
)

// this will iterate through all of normandy's recipes and product this kind of output:
//
// Date			Type				Created			Updated			Paused
// ----    		____				-------			-------			------
// 2020-07-07	recipe-type				  5				  3				 7
// 2020-07-07	recipe-type				  5				  3				 7
// 2020-07-07	recipe-type				  5				  3				 7
// 2020-07-06	recipe-type				  5				  3				 7
// 2020-07-06	recipe-type				  5				  3				 7
//
//
// Date
// Type
// Created
// Updated
// Paused

type DataRecord struct {
	Date       string
	RecipeType string
	Created    int
	Updated    int
	Paused     int
}

func key(r *Record) string {
	return r.Date + "-" + r.RecipeType
}

func main() {

	data := make([string]*DataRecord)

	baseUrl := "https://normandy.cdn.mozilla.net/api/v3/recipe/"

	// lots of workers to load and process data fast
	workChan := make(chan int, 8)
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

					type HistoryRecord struct {
						ActionType string
						Date       string
						Enabled    bool
					}

					history := make([]HistoryRecord, 0, 8)

					jsonparser.ArrayEach(body, func(value []byte, _ jsonparser.ValueType, _ int, _ error) {
						created, _ := jsonparser.GetString(value, "date_created")
						enabled, _ := jsonparser.GetBoolean(value, "enabled")

						// extract the info
						history = append(history, HistoryRecord{created, enabled})
					})

					// iterate through history in reverse order, api returns things chronologically
					last := len(history) - 1
					for i := last; i >= 0; i-- {
						if i == last {
							// increment the date

						}

					}

					// put it back (struct, not a pointer ...)
					revisions.Set(id, record)
				}
			}
		}()
	}

	next := baseUrl
	count := 0
	for {
		// useful when hacking to not load allllll the pages
		if count++; count > 10 {
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

		// don't worry about the error, blank string will break the loop
		next, _ = jsonparser.GetString(body, "next")

		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			id64, err := jsonparser.GetInt(value, "id")
			if err != nil {
				fmt.Println("Error extracting id: ", err.Error())
				return
			}

			// convert it to an int
			id := int(id64)
			workChan <- id

		}, "results")
	}

	// send a stop signal
	workChan <- -1
	wg.Wait()

	fmt.Println("Done")
}
