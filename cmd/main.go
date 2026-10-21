package main

import (
	"log"
	"priceManager/configreader"
	"priceManager/database"
	"priceManager/srv"
)

func main() {
	data, err := configreader.CheckConfigFile("config.yaml")
	if err != nil {
		log.Println(err)
		return
	}
	cfg, err := configreader.Parse(data)
	if err != nil {
		log.Println(err)
		return
	}

	db, err := database.Init()
	if err != nil {
		log.Println(err)
		return
	}
	err = database.Migrate(db.DB)
	if err != nil {
		log.Println()
	}
	log.Println(srv.New(db, cfg.HTTP.Port).Run(cfg.Root.Email, cfg.Root.Password, cfg.HTTP.ServerIP, cfg.CompanyName, cfg.HTTP.ServerDNSN))
}
