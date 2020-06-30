package main

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/mostlygeek/normandy-tools/tools"
)

func main() {

	baseUrl := "https://normandy.cdn.mozilla.net/api/v3/recipe/"
	next := baseUrl + "?ordering=latest_revision"

	for {
		if next == "" {
			break
		}

		//fmt.Println("Fetching:", next)
		// hmm
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
			id, err := jsonparser.GetInt(value, "id")
			if err != nil {
				fmt.Println("Error extracting id: ", err.Error())
				return
			}

			created, err := jsonparser.GetString(value, "latest_revision", "date_created")
			if err != nil {
				fmt.Println("Iteration error", err.Error())
				return
			}

			// Filter out all the heartbeat messages
			actionType, _ := jsonparser.GetString(value, "latest_revision", "action", "name")
			slug, _ := jsonparser.GetString(value, "latest_revision", "arguments", "slug")
			fmt.Println(created[0:10], id, actionType, slug)

		}, "results")
	}
}
