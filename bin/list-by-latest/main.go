package main

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/mostlygeek/normandy-tools/tools"
)

func main() {

	baseUrl := "https://normandy.cdn.mozilla.net/api/v3/recipe/"
	next := baseUrl + "?ordering=latest_revision"

	tools.WalkAPI(next, func(record []byte) error {
		id, err := jsonparser.GetInt(record, "id")
		if err != nil {
			fmt.Println("Error extracting id: ", err.Error())
			return nil
		}

		created, err := jsonparser.GetString(record, "latest_revision", "date_created")
		if err != nil {
			fmt.Println("Iteration error", err.Error())
			return nil
		}

		// Filter out all the heartbeat messages
		actionType, _ := jsonparser.GetString(record, "latest_revision", "action", "name")
		slug, _ := jsonparser.GetString(record, "latest_revision", "arguments", "slug")
		fmt.Println(created[0:10], id, actionType, slug)

		return nil
	})
}
