package main

import (
	"fmt"
	"os"
	"os/signal"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("TOKEN")
	mysql := os.Getenv("MYSQL")

	db, err := sql.Open("mysql", mysql)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	d, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	err = d.Open()
	if err != nil {
		panic(err)
	}
	defer d.Close()

	u, err := d.User("@me")
	if err != nil {
		panic(err)
	}

	botID := u.ID

	d.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == botID {
			return
		}

		if m.Content == ".games" {
			var (
				game string
				creator string
				slotsTaken int
				slotsTotal int
				totalGames int
			)

			rows, err := db.Query("SELECT gamename, creatorname, slotstaken, slotstotal, totalgames FROM gamelist");
			gameExists := false
			if err != nil {
				panic(err)
			}
			defer rows.Close()
			for rows.Next() {
				err := rows.Scan(&game, &creator, &slotsTaken, &slotsTotal, &totalGames);
				if err != nil {
					panic(err)
				}
				var format string
				if totalGames == 0 {
					format = "Game [%s : %s : %d/%d] is in the lobby"
				} else {
					format = "Game [%s : %s : %d/%d] is in progress"
				}
				msg := fmt.Sprintf(format, game, creator, slotsTaken, slotsTotal)
				s.ChannelMessageSend(m.ChannelID, msg)
				gameExists = true
			}
			err = rows.Err()
			if err != nil {
				panic(err)
			}
			if !gameExists {
				s.ChannelMessageSend(m.ChannelID, "No games available :crying_cat_face:")
			}

		}
	})

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
