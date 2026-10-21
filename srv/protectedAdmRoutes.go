package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"priceManager/database"
	"strconv"
	"strings"
)

// регистрация маршрутов группы protect
func (srv *ServerHTTP) registerProtectedAdminRoutes() {
	protectedAdmRoutes := http.NewServeMux()
	//admin workspace
	protectedAdmRoutes.HandleFunc("/adm/ws", showAdminWorkspace)
	//rooms
	protectedAdmRoutes.HandleFunc("POST /adm/api/rooms", srv.addRoom)
	protectedAdmRoutes.HandleFunc("PATCH /adm/api/rooms/{id}", srv.renameRoom)
	protectedAdmRoutes.HandleFunc("DELETE /adm/api/rooms/{id}", srv.deleteRoom)
	//slots
	protectedAdmRoutes.HandleFunc("POST /adm/api/rooms/{id}/slots", srv.addSlot)
	protectedAdmRoutes.HandleFunc("DELETE /adm/api/slots/{id}", srv.deleteSlot)
	protectedAdmRoutes.HandleFunc("POST /adm/api/slots/{id}/image", srv.uploadImage)
	//start protected handler
	srv.mux.Handle("/adm/", mv(protectedAdmRoutes))
}

// middleware для проверки легитимности запросов по маршрутам группы
func mv(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("ssID")
		if err != nil {
			http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
			return
		}
		if c.Value != "osfr" {
			http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// отдает админскую страницу
func showAdminWorkspace(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("./http/adm/adm.html")
	if err != nil {
		http.Error(w, err.Error(), 404)
	}
	w.Write(data)
}

// добавляет комнату на сервер
func (srv *ServerHTTP) addRoom(w http.ResponseWriter, r *http.Request) {
	newRoom := Room{}
	if err := json.NewDecoder(r.Body).Decode(&newRoom); err != nil {
		log.Println(err)
		http.Error(w, "room appending error", http.StatusInternalServerError)
		return
	}

	dataRoom := database.Room{Title: newRoom.Title}

	if err := srv.dbConn.Rooms().Add(context.Background(), &dataRoom); err != nil {
		log.Println(err)
		http.Error(w, "room creating error", http.StatusInternalServerError)
		return
	}

	newRoom.Id = dataRoom.ID
	newRoom.Title = dataRoom.Title

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(newRoom); err != nil {
		log.Println(err)
		http.Error(w, "new room data response sent error", http.StatusInternalServerError)
		return
	}
}

// обновляет название комнаты на сервере
func (srv *ServerHTTP) renameRoom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "target room id parsing error", http.StatusInternalServerError)
		return
	}

	room := database.Room{}

	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		log.Println(err)
		http.Error(w, "room title parsing error", http.StatusInternalServerError)
		return
	}

	room.ID = uint(id)

	if err := srv.dbConn.Rooms().Rename(context.Background(), &room); err != nil {
		log.Println(err)
		http.Error(w, "room updating error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(Room{Id: room.ID, Title: room.Title}); err != nil {
		log.Println(err)
		http.Error(w, "room data encoding error", http.StatusInternalServerError)
		return
	}
}

// удаляет комнату из базы данных
func (srv *ServerHTTP) deleteRoom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "id parsing error", http.StatusInternalServerError)
		return
	}

	if _, err := srv.dbConn.Rooms().Delete(context.Background(), uint(id)); err != nil {
		log.Println(err)
		http.Error(w, "room deleting error", http.StatusInternalServerError)
		return
	}
}

// хандлер
// нужен для загрузки img на сервер
func (srv *ServerHTTP) uploadImage(w http.ResponseWriter, r *http.Request) {
	const maxUpload = 10 << 20

	// 1. Ограничение всего тела запроса
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+1<<20)

	// 2. Парсим multipart, много в память не тянем
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad id"})
		return
	}

	// 3. Проверяем, что файл один
	if r.MultipartForm != nil && len(r.MultipartForm.File["image"]) > 1 {
		http.Error(w, "можно загрузить только один файл", 400)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "нет поля 'image' с файлом", 400)
		return
	}
	defer file.Close()

	// 4. Проверяем размер ещё раз (после парсинга header.Size известен)
	if header.Size > maxUpload {
		http.Error(w, "файл больше 10 МБ", 400)
		return
	}

	// 5. Проверяем реальный тип по magic bytes
	head := make([]byte, 512)
	n, _ := file.Read(head)
	file.Seek(0, io.SeekStart) // вернуть указатель в начало
	mime := http.DetectContentType(head[:n])
	if !strings.HasPrefix(mime, "image/") {
		http.Error(w, "не изображение", 400)
		return
	}

	slotID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "id is not number", 500)
		log.Println(fmt.Errorf("id number parsing error: %w", err))
		return
	}

	slot, err := srv.dbConn.Slots().GetByID(context.Background(), uint(slotID))
	if err != nil {
		log.Println(fmt.Errorf("slot image updating: slot data getting error::%w", err))
		return
	}

	// 6. Сохраняем
	slot.ImagePath = filepath.Join("papkaDLyaIMG", strconv.FormatUint(uint64(slot.ID), 10))

	_, err = srv.dbConn.Slots().UpdateImage(context.Background(), &slot)
	if err != nil {
		err = fmt.Errorf("cannot save slot image in db: %w", err)
		log.Println(err)
		return
	}

	fmt.Printf("%+v\n", slot)
}

// добавляет слот в комнату
func (srv *ServerHTTP) addSlot(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "room id parsing error", http.StatusInternalServerError)
		return
	}
	slotData := database.Slot{RoomID: uint(roomId)}
	if err := srv.dbConn.Slots().AddByRoom(context.Background(), &slotData); err != nil {
		log.Println(err)
		http.Error(w, "slot creating error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(Slot{Id: slotData.ID, ImageUrl: slotData.ImagePath})

}

// удаляет слот
func (srv *ServerHTTP) deleteSlot(w http.ResponseWriter, r *http.Request) {
	slotID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "solt id parsing error", http.StatusInternalServerError)
		return
	}

	if _, err := srv.dbConn.Slots().Delete(context.Background(), uint(slotID)); err != nil {
		log.Println(err)
		http.Error(w, "slot deleting error", http.StatusInternalServerError)
		return
	}
}
