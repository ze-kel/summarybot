package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	mdbot "github.com/ze-kel/summarybot/cmd/bot"
	"github.com/ze-kel/summarybot/cmd/db"
)

func init() {
	// load .env
	if err := godotenv.Load(); err != nil {
		log.Print("WARN: No .env file found. This is fine if you are running inside docker container.")
	}
}

func initBot() {
	token, exists := os.LookupEnv("TG_TOKEN")
	if !exists {
		panic("NO TG_TOKEN IN ENV")
	}

	pgUrl, isPg := os.LookupEnv("POSTGRES_URL")

	if isPg {
		log.Printf("POSTGRES_URL is set, connecting to external db")
	} else {
		log.Print("POSTGRES_URL is not set, using local sqlite")
	}

	OPENAI_TOKEN, exists := os.LookupEnv("OPENAI_KEY")
	if !exists {
		panic("NO OPENAI_KEY IN ENV")
	}

	client := openai.NewClient(OPENAI_TOKEN)

	database := db.Init(pgUrl)

	m := mdbot.New(database, token, client)

	m.Start()
}

func main() {
	initBot()
}
