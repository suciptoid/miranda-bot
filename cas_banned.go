package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func checkBanned(id int64) bool {
	resp, err := http.Get(fmt.Sprintf("https://api.cas.chat/check?user_id=%d", id))
	if err != nil {
		log.Printf("[cas] unable check user status on CAS banned, status: %d", resp.StatusCode)
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
