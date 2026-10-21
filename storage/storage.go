package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Storage struct {
	rootDir string
}

func New(rootDir string) *Storage {
	return &Storage{rootDir: rootDir}
}

// EnsureRoomDir создает папку для комнаты, если ее нет.
func (s *Storage) EnsureRoomDir(roomID uint) (string, error) {
	dir := filepath.Join(s.rootDir, fmt.Sprintf("%d", roomID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create room dir: %w", err)
	}
	return dir, nil
}

// SaveBlob сохраняет картинку из веб-запроса на диск.
// fileName — имя файла (например, сгенерированный UUID + .jpg).
func (s *Storage) SaveBlob(roomID uint, fileName string, data io.Reader) (string, error) {
	dir, err := s.EnsureRoomDir(roomID)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(dir, fileName)
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, data); err != nil {
		_ = os.Remove(fullPath) // откат при ошибке
		return "", fmt.Errorf("write file: %w", err)
	}

	return fullPath, nil
}

// ReadBlob читает файл с диска и возвращает его содержимое как []byte.
func (s *Storage) ReadBlob(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", filePath, err)
	}
	return data, nil
}

// DeleteBlob удаляет файл с диска.
func (s *Storage) DeleteBlob(filePath string) error {
	if filePath == "" {
		return nil
	}
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

// удаляет файлы на сервере
func (s *Storage) DeleteRoomDir(roomID uint) error {
	dir := filepath.Join(s.rootDir, strconv.FormatUint(uint64(roomID), 10))
	return os.RemoveAll(dir) // RemoveAll не падает, если папки нет
}
