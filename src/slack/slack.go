package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func SendChannelMessage(hookUrl, msg string) error {
	reqStruct := struct {
		Text string `json:"text"`
	}{Text: msg}

	data, err := json.Marshal(reqStruct)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", hookUrl, bytes.NewReader(data))
	if err != nil {
		return nil
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 299 {
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Println("response Body:", string(body))
		return errors.New("Received non-200 response " + response.Status + " body: " + string(body))
	}
	return nil
}
