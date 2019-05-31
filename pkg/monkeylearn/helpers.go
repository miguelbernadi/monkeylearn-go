package monkeylearn

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func startTimer(name string) func() {
	t := time.Now()
	return func() {
		d := time.Since(t)
		log.Println(name, "took", d)
	}
}

func deserializeResponse(response *http.Response) ([]Result, error) {
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var res []Result
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
