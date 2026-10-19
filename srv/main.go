package srv

import (
	"context"
	"fmt"
	"net/http"
	"priceManager/database"
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
	}

	srv.registerProtectedAdminRoutes()
	srv.registerPublicRoutes()

	return srv
}

func (srv *ServerHTTP) Run(rootEmail, rootPassword string) error {
	if err := srv.dbConn.Users().SetAsRoot(context.Background(), rootEmail, rootPassword); err != nil {
		return fmt.Errorf("http server run preparing error: %w", err)
	}
	fmt.Printf("server started with root credentials from config file - config.yaml\n rootEmail: %s\n rootPassword: %s\nIf you will this credentials change - update config.yaml please\n", rootEmail, rootPassword)
	return srv.server.ListenAndServe()
}
