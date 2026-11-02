package srv

import (
	"log"
	"os"
)

// NewLogger создает единый логгер для всего приложения.
// Он пишет в стандартный поток ошибок (stderr), что позволяет легко
// перенаправлять логи в файл средствами ОС (например, > file 2>&1).
func NewLogger() *log.Logger {
	// log.LstdFlags добавляет дату и время (2026/11/02 10:30:00)
	// Если хотите видеть файл и строку, добавьте log.Lshortfile
	return log.New(os.Stderr, "", log.LstdFlags)
}
