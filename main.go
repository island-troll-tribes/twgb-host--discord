package main

import (
	"os"
	"log"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("TOKEN")
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("error creating session ", err)
	}

	u, err := d.User("@me")
	if err != nil {
		log.Fatal("error fetching user ", err)
	}

	botID := u.ID

	d.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == botID {
			return
		}

		if m.Content == ".games" {
			s.ChannelMessageSend(m.ChannelID, "Pong!")
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi!")
	})
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
