package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func (srv *ServerHTTP) registerPublicRoutes() {
	srv.mux.HandleFunc(" /", mainPage)
	srv.mux.HandleFunc("GET /api/rooms", listRooms)
	srv.mux.HandleFunc("GET /api/rooms/{id}", getRoom)
	srv.mux.HandleFunc(" /auth", authentificationPage)
	srv.mux.HandleFunc("POST /auth/check", srv.setAuth)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("./http/public/main.html")
	if err != nil {
		err = fmt.Errorf("public/main.html opening error: %w", err)
		http.Error(w, "no content", http.StatusNotFound)
		return
	}

	w.Write(data)
}

// отдает список комнат json
func listRooms(w http.ResponseWriter, r *http.Request) {
	rooms := make([]Room, 2)
	rooms[0] = Room{1, "Розница"}
	rooms[1] = Room{2, "Парки"}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(rooms)
}

// отдает содержимое комнаты json
func getRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "1" {
		slots1 := make([]Slot, 1)
		slots1[0] = Slot{Id: 1, ImageUrl: "", SortOrder: 1}
		slots := SlotResponse{Id: 1, Title: "Розница", Slots: slots1}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(slots)
	}
}

func authentificationPage(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("./http/adm/auth.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	w.Write(data)
}

func (srv *ServerHTTP) setAuth(w http.ResponseWriter, r *http.Request) {
	type Credentials struct {
		login, passwd string
	}

	creds := Credentials{login: r.FormValue("login"), passwd: r.FormValue("password")}

	userMGR := srv.dbConn.Users()

	usr, err := userMGR.GetByCredentials(context.Background(), creds.login, creds.passwd)
	if err != nil {
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
		return
	}

	usr.AuthHash = "osfr"

	if err := userMGR.UpdateUser(context.Background(), usr); err != nil {
		log.Println(err)
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "ssID",
		Value:    usr.AuthHash,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // true на HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()), // 24 часа
	})

	http.Redirect(w, r, "/adm/ws", http.StatusTemporaryRedirect)
}
