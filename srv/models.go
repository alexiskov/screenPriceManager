package srv

import (
	"net/http"
	"priceManager/database"
)

type (
	ServerHTTP struct {
		server *http.Server
		mux    *http.ServeMux
		dbConn *database.Storage
	}

	Room struct {
		Id    uint   `json:"id"`
		Title string `json:"title"`
	}

	Slot struct {
		Id        uint   `json:"id"`
		ImageUrl  string `json:"image_url"`
		SortOrder int    `json:"sort_order"`
	}

	SlotResponse struct {
		Room
		Slots []Slot `json:"slots"`
	}
)
