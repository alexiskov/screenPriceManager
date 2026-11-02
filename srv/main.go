package srv

import (
	"context"
	"fmt"
	"net/http"
	"priceManager/database"
	"priceManager/storage"
	"time"
)

// создает сервер
func New(db *database.Storage, port string) *ServerHTTP {
	mux := http.NewServeMux()
	logger := NewLogger()

	srv := &ServerHTTP{
		mux:    mux,
		dbConn: db,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%s", port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second, // Критично для защиты от Slowloris
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
			ErrorLog:          logger,
		},
		storage: storage.New("uploads"),
		logger:  logger,
	}

	srv.registerProtectedAdminRoutes()
	srv.registerPublicRoutes()

	return srv
}

// Run запускает сервер по HTTPS, автоматически создавая папку и сертификаты.
func (srv *ServerHTTP) Run(rootEmail, rootPassword, ip, organization, dnsn string) {
	if err := srv.dbConn.Users().SetAsRoot(context.Background(), rootEmail, rootPassword); err != nil {
		srv.logger.Println(fmt.Errorf("http server run preparing error: %w", err))
		return
	}

	// Подготавливаем сертификаты (папка certs/ и файлы cert.pem/key.pem)
	certPath, keyPath, err := ensureCertificates(ip, organization, dnsn)
	if err != nil {
		srv.logger.Println((fmt.Errorf("certificates error: %w", err)))
		return
	}

	srv.logger.Printf("server started with root credentials from config file - config.yaml\n rootEmail: %s\n rootPassword: %s\nIf you will this credentials change - update config.yaml please\n", rootEmail, rootPassword)
	srv.logger.Printf("🚀 Server running on https://%s:%s\n", dnsn, srv.server.Addr)

	// Запускаем HTTPS
	srv.logger.Println(srv.server.ListenAndServeTLS(certPath, keyPath))
}
