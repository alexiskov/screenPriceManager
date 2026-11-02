package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"priceManager/database"
	"strconv"
	"strings"
	"time"
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
		srv.logger.Println(err)
		http.Error(w, "room appending error", http.StatusInternalServerError)
		return
	}

	dataRoom := database.Room{Title: newRoom.Title}

	if err := srv.dbConn.Rooms().Add(context.Background(), &dataRoom); err != nil {
		srv.logger.Println(err)
		http.Error(w, "room creating error", http.StatusInternalServerError)
		return
	}

	newRoom.Id = dataRoom.ID
	newRoom.Title = dataRoom.Title

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(newRoom); err != nil {
		srv.logger.Println(err)
		http.Error(w, "new room data response sent error", http.StatusInternalServerError)
		return
	}
}

// обновляет название комнаты на сервере
func (srv *ServerHTTP) renameRoom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.logger.Println(err)
		http.Error(w, "target room id parsing error", http.StatusInternalServerError)
		return
	}

	room := database.Room{}

	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		srv.logger.Println(err)
		http.Error(w, "room title parsing error", http.StatusInternalServerError)
		return
	}

	room.ID = uint(id)

	if err := srv.dbConn.Rooms().Rename(context.Background(), &room); err != nil {
		srv.logger.Println(err)
		http.Error(w, "room updating error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(Room{Id: room.ID, Title: room.Title}); err != nil {
		srv.logger.Println(err)
		http.Error(w, "room data encoding error", http.StatusInternalServerError)
		return
	}
}

// удаляет комнату из базы данных
func (srv *ServerHTTP) deleteRoom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.logger.Println(err)
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	// 1. Удаляем комнату и её слоты из БД (ваш метод RoomsData.Delete)
	if _, err := srv.dbConn.Rooms().Delete(context.Background(), uint(id)); err != nil {
		srv.logger.Println(err)
		http.Error(w, "room deleting error", http.StatusInternalServerError)
		return
	}

	// 2. Удаляем папку со всеми картинками этой комнаты
	// (Метод DeleteRoomDir мы добавили в пакет storage на предыдущем шаге)
	if err := srv.storage.DeleteRoomDir(uint(id)); err != nil {
		// Если удалить папку не удалось, БД уже очищена, поэтому не возвращаем 500.
		// Просто логируем, чтобы можно было почистить вручную.
		srv.logger.Printf("warning: failed to delete dir for room %d: %v", id, err)
	}

	// 3. Отвечаем фронтенду, что всё ок
	w.WriteHeader(http.StatusNoContent)
}

// хандлер
// нужен для загрузки img на сервер
func (srv *ServerHTTP) uploadImage(w http.ResponseWriter, r *http.Request) {
	const maxUpload = 10 << 20

	// 1. Ограничение всего тела запроса
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+1<<20)

	// 2. Парсим multipart
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		srv.logger.Println(fmt.Errorf("from address %s, error:%w", r.RemoteAddr, err))
		http.Error(w, "bad form data", http.StatusBadRequest)
		return
	}

	// 3. Проверяем, что файл один
	if r.MultipartForm != nil && len(r.MultipartForm.File["image"]) > 1 {
		http.Error(w, "можно загрузить только один файл", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		srv.logger.Println(fmt.Errorf("from address %s, error:%w", r.RemoteAddr, err))
		http.Error(w, "нет поля 'image' с файлом", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 4. Проверяем размер ещё раз
	if header.Size > maxUpload {
		http.Error(w, "файл больше 10 МБ", http.StatusBadRequest)
		return
	}

	// 5. Проверяем реальный тип по magic bytes
	head := make([]byte, 512)
	n, _ := file.Read(head)
	file.Seek(0, io.SeekStart)
	mime := http.DetectContentType(head[:n])
	if !strings.HasPrefix(mime, "image/") {
		http.Error(w, "не изображение", http.StatusBadRequest)
		return
	}

	slotID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		srv.logger.Println(fmt.Errorf("from address %s, slorID parsing error:%w", r.RemoteAddr, err))
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	slot, err := srv.dbConn.Slots().GetByID(context.Background(), uint(slotID))
	if err != nil {
		http.Error(w, "slot not found", http.StatusNotFound)
		return
	}

	// Генерируем уникальное имя
	fileName := fmt.Sprintf("img_%d_%d.jpg", slot.ID, time.Now().UnixNano())

	// Сохраняем поток байт на диск
	savedPath, err := srv.storage.SaveBlob(slot.RoomID, fileName, file)
	if err != nil {
		srv.logger.Println("save file error:", err)
		http.Error(w, "cannot save file", http.StatusInternalServerError)
		return
	}

	// Обновляем БД
	oldPath := slot.ImagePath
	slot.ImagePath = savedPath // например, "uploads/1/img_1_123.jpg"

	// ВОТ ЗДЕСЬ ПРОВЕРЯЕМ РЕЗУЛЬТАТ
	rowsAffected, err := srv.dbConn.Slots().UpdateImage(context.Background(), &slot)
	if err != nil {
		_ = srv.storage.DeleteBlob(savedPath) // откат
		srv.logger.Println("db update error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		_ = srv.storage.DeleteBlob(savedPath) // откат
		srv.logger.Println("db update error: rows affected = 0 (slot not found?)")
		http.Error(w, "slot not updated", http.StatusInternalServerError)
		return
	}

	// Удаляем старую картинку, если она была
	if oldPath != "" {
		_ = srv.storage.DeleteBlob(oldPath)
	}

	// ОТПРАВЛЯЕМ ОТВЕТ ФРОНТЕНДУ
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        slot.ID,
		"image_url": fmt.Sprintf("/api/slots/%d/image", slot.ID), // Виртуальный URL
	})
}

// добавляет слот в комнату
func (srv *ServerHTTP) addSlot(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.logger.Println(err)
		http.Error(w, "room id parsing error", http.StatusInternalServerError)
		return
	}
	slotData := database.Slot{RoomID: uint(roomId)}
	if err := srv.dbConn.Slots().AddByRoom(context.Background(), &slotData); err != nil {
		srv.logger.Println(err)
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
		srv.logger.Println(err)
		http.Error(w, "slot id parsing error", http.StatusBadRequest)
		return
	}
	slotMGR := srv.dbConn.Slots()
	slot, err := slotMGR.GetByID(context.Background(), uint(slotID))
	if err != nil {
		srv.logger.Println(err)
		http.Error(w, "slot deleting error", http.StatusInternalServerError)
		return
	}
	if err := srv.storage.DeleteBlob(slot.ImagePath); err != nil {
		srv.logger.Println(err)
		http.Error(w, "image slot deleting error", http.StatusInternalServerError)
		return
	}

	if rows, err := slotMGR.Delete(context.Background(), uint(slotID)); err != nil {
		srv.logger.Println(err)
		http.Error(w, "slot deleting error", http.StatusInternalServerError)
		return
	} else {
		// Если rows == 0, значит запись уже кто-то удалил параллельно (гонка)
		// Это не ошибка для клиента, просто выходим с 204.
		if rows == 0 {
			srv.logger.Printf("deleteSlot: slot %d already deleted", slotID)
		}

		// 5. Возвращаем 204 No Content (успех, тела нет)
		w.WriteHeader(http.StatusNoContent)
	}

}
