package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

var casBaseURL = "https://api.cas.chat/check"

func checkBanned(id int64) bool {
	resp, err := http.Get(fmt.Sprintf("%s?user_id=%d", casBaseURL, id))
	if err != nil {
		log.Printf("[cas] unable check user status on CAS banned, error: %v", err)
		return false
	}

	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)

	var cas struct {
		OK bool `json:"ok"`
	}

	json.Unmarshal(bytes, &cas)
	log.Printf("[cas] result from CAS API %v", cas.OK)
	return cas.OK
}
