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
	"strconv"
	"strings"
)

// регистрация маршрутов группы protect
func (srv *ServerHTTP) registerProtectedAdminRoutes() {
	protectedAdmRoutes := http.NewServeMux()

	protectedAdmRoutes.HandleFunc("/adm/ws", showAdminWorkspace)

	protectedAdmRoutes.HandleFunc("POST /adm/api/slots/{id}/image", srv.uploadImage)

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
	dir := filepath.Join("papkaDLyaIMG", strconv.FormatUint(uint64(slot.ID), 10))
	slot.ImagePath = dir

	_, err = srv.dbConn.Slots().UpdateImage(context.Background(), &slot)
	if err != nil {
		err = fmt.Errorf("cannot save slot image in db: %w", err)
		log.Println(err)
		return
	}

	fmt.Printf("%+v\n", slot)
}
