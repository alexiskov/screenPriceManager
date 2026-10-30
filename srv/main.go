package srv

import (
	"context"
	"fmt"
	"net/http"
	"priceManager/database"
	"priceManager/storage"
	"time"
)

func New(db *database.Storage, port string) *ServerHTTP {
	mux := http.NewServeMux()

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
		},
		storage: storage.New("uploads"),
	}

	srv.registerProtectedAdminRoutes()
	srv.registerPublicRoutes()

	return srv
}

// Run запускает сервер по HTTPS, автоматически создавая папку и сертификаты.
func (srv *ServerHTTP) Run(rootEmail, rootPassword, ip, organization, dnsn string) error {
	if err := srv.dbConn.Users().SetAsRoot(context.Background(), rootEmail, rootPassword); err != nil {
		return fmt.Errorf("http server run preparing error: %w", err)
	}

	// Подготавливаем сертификаты (папка certs/ и файлы cert.pem/key.pem)
	certPath, keyPath, err := ensureCertificates(ip, organization, dnsn)
	if err != nil {
		return fmt.Errorf("certificates error: %w", err)
	}

	fmt.Printf("server started with root credentials from config file - config.yaml\n rootEmail: %s\n rootPassword: %s\nIf you will this credentials change - update config.yaml please\n", rootEmail, rootPassword)
	fmt.Printf("🚀 Server running on https://%s:%s\n", dnsn, srv.server.Addr)

	// Запускаем HTTPS
	return srv.server.ListenAndServeTLS(certPath, keyPath)
}
