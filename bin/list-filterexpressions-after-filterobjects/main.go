package main

import (
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mostlygeek/normandy-tools/tools"
)

// goes through and dumps an easier to see pattern of filter expressions we write
// that can't be handled in pure FilterObjects.  The output can be found in this gist:
//
// https://gist.github.com/mostlygeek/875b3a9a0127b2f046c21a6c62d9584e
//
// We are starting to turn complex, repeating experiment targeting with the new "preset_choices"
// FilterObject.  Not sure if we are continue running complex experiments like this ...

func main() {

	baseUrl := "https://normandy.cdn.mozilla.net/api/v3/recipe/"

	tools.WalkAPI(baseUrl, func(record []byte) error {
		id, err := jsonparser.GetInt(record, "id")
		if err != nil {
			fmt.Println("Error extracting id: ", err.Error())
			return err
		}

		created, err := jsonparser.GetString(record, "latest_revision", "date_created")
		if err != nil {
			fmt.Println("Iteration error", err.Error())
			return err
		}

		actionType, _ := jsonparser.GetString(record, "latest_revision", "action", "name")

		if actionType == "show-heartbeat" {
			return nil
		}

		extra_fo, _ := jsonparser.GetString(record, "latest_revision", "extra_filter_expression")
		extra_fo = strings.Replace(extra_fo, "\n", "", -1)
		extra_fo = strings.Replace(extra_fo, " ", "", -1)

		if extra_fo == "" {
			return nil
		}

		fmt.Printf("%s %d %s [%s]\n", created[0:10], id, actionType, extra_fo)

		return nil
	})
}
