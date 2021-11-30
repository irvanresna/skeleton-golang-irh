package handler

import (
	"net/http"
)

// Test ...
func (h Contract) Test(w http.ResponseWriter, r *http.Request) {

	h.SendSuccess(w, nil, nil)
}
