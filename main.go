package main

import (
	"os"
	"os/signal"
	"database/sql"
	"log"
	"fmt"
	"time"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/bwmarrin/discordgo"
)

type Game struct {
	Name string
	Creator string
	SlotsTaken int
	SlotsTotal int
	InProgress bool
}

func main() {
	token := os.Getenv("TOKEN")
	mysql := os.Getenv("MYSQL")
	defaultChannelID := os.Getenv("DEFAULT_CHANNEL_ID")
	period, err := strconv.Atoi(os.Getenv("POLLING_PERIOD"))
	production := os.Getenv("GO_ENV") == "production"

	log := log.New(os.Stderr, "TwGB[HOST] ", log.Ldate | log.Ltime | log.Lshortfile)
	games := map[int]Game{}

	if err != nil {
		period = 1
	}

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

	ticker := time.NewTicker(time.Duration(period) * time.Second)
	go func() {
		for {
			<-ticker.C

			gameExists := map[int]bool{}
			rows, err := db.Query("SELECT id, gamename, creatorname, slotstaken, slotstotal, totalgames FROM gamelist")
			if err != nil {
				panic(err)
			}
			defer rows.Close()

			for rows.Next() {
				var (
					id, totalGames int
					msg string
					game Game
				)

				err := rows.Scan(&id, &game.Name, &game.Creator, &game.SlotsTaken, &game.SlotsTotal, &totalGames)
				if err != nil {
					panic(err)
				}
				game.InProgress = totalGames == 1
				gameExists[id] = true

				if _, ok := games[id]; ok {
					if !games[id].InProgress && game.InProgress {
						msg = fmt.Sprintf("Game started [%s : %s : %d/%d] :palm_tree: @disciples", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal)
						if production {
							d.ChannelMessageSend(defaultChannelID, msg)
						}
						log.Print(msg)
					}
				} else {
					if game.InProgress {
						msg = fmt.Sprintf("Game in progress [%s : %s : %d/%d] :smiley_cat: @disciples", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal)
						if production {
							d.ChannelMessageSend(defaultChannelID, msg)
						}
						log.Print(msg)
					} else {
						msg = fmt.Sprintf("New game [%s : %s : %d/%d] :smiley_cat: @disciples", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal)
						if production {
							d.ChannelMessageSend(defaultChannelID, msg)
						}
						log.Print(msg)
					}
				}

				games[id] = game
			}

			err = rows.Err()
			if err != nil {
				panic(err)
			}

			for id, game := range(games) {
				if !gameExists[id] {
					var msg string
					if game.InProgress {
						msg = fmt.Sprintf("Game over [%s : %s : %d/%d] :fire: @disciples", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal)
					} else {
						msg = fmt.Sprintf("Lobby ended [%s : %s : %d/%d] :dash: @disciples", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal)
					}
					if production {
						d.ChannelMessageSend(defaultChannelID, msg)
					}
					log.Print(defaultChannelID, " ", msg)
					delete(games, id)
				}
			}
		}
	}()

	d.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		switch m.Content {
		case ".games":
			for _, game := range(games) {
				var format string
				if game.InProgress {
					format = "Game [%s : %s : %d/%d] is in progress @disciples"
				} else {
					format = "Game [%s : %s : %d/%d] is in the lobby @disciples"
				}
				msg := fmt.Sprintf(format, game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal)
				if production {
					d.ChannelMessageSend(m.ChannelID, msg)
				}
				log.Print(m.ChannelID, " ", msg)
			}
			if len(games) == 0 {
				if production {
					s.ChannelMessageSend(m.ChannelID, "No games available :crying_cat_face: @disciples")
				}
				log.Print(m.ChannelID, " 128 ", "No games available :crying_cat_face: @disciples")
			}
		case ".subscribe":
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				log.Print(err)
				break
			}
			err = s.GuildMemberRoleAdd(channel.GuildID, m.Author.ID, "disciples")
			if err != nil {
				log.Print(err)
			}
		case ".unsubscribe":
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				log.Print(err)
				break
			}
			err = s.GuildMemberRoleRemove(channel.GuildID, m.Author.ID, "disciples")
			if err != nil {
				log.Print(err)
			}
		}
	})

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
