package main

import (
	"log"
	"net/http"

	"github.com/insider/football-league/internal/db"
	"github.com/insider/football-league/internal/handlers"
	"github.com/insider/football-league/internal/service"
)

func main() {
	database, err := db.Open("./league.db", "./schema/schema.sql")
	if err != nil {
		log.Fatal("veritabanı açılamadı:", err)
	}
	defer database.Close()

	leagueService, err := service.NewLeagueManager(database)
	if err != nil {
		log.Fatal("servis başlatılamadı:", err)
	}

	apiHandler := handlers.New(leagueService)

	log.Println("Sunucu http://localhost:8080 adresinde çalışıyor...")
	err = http.ListenAndServe(":8080", apiHandler.Routes())
	if err != nil {
		log.Fatal("sunucu çöktü:", err)
	}
}
