package utils

import (
	"encoding/json"
	"net/http"
)

func SendJSONReplyOK(w http.ResponseWriter, replyContent interface{}) {
	SendJSONReplyStatus(w, http.StatusOK, replyContent)
}

func SendJSONReplyStatus(w http.ResponseWriter, status int, replyContent interface{}) {
	toSend, err := json.Marshal(replyContent)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(status)
	_, err = w.Write(toSend)
	if err != nil {
		panic(err)
	}
}
