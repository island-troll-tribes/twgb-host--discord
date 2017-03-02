package main

import (
	"os"
	"os/signal"
	"database/sql"
	"log"
	"fmt"
	"time"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/bwmarrin/discordgo"
)

type Game struct {
	Name string
	Creator string
	Players []string
	SlotsTaken int
	SlotsTotal int
	InProgress bool
}

func (g *Game) PlayerNames() string {
	return strings.Join(g.Players, ", ")
}

func main() {
	token := os.Getenv("TOKEN")
	mysql := os.Getenv("MYSQL")
	subscriberRole := os.Getenv("SUBSCRIBER_ROLE")
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
			rows, err := db.Query("SELECT id, gamename, creatorname, slotstaken, slotstotal, totalgames, usernames FROM gamelist")
			if err != nil {
				panic(err)
			}
			defer rows.Close()

			for rows.Next() {
				var (
					id, totalGames int
					msg, usernames string
					game Game
				)

				err := rows.Scan(&id, &game.Name, &game.Creator, &game.SlotsTaken, &game.SlotsTotal, &totalGames, &usernames)
				if err != nil {
					panic(err)
				}
				game.InProgress = totalGames == 1
				gameExists[id] = true

				parts := strings.Split(usernames, "\t")
				game.Players = []string{}
				for i, part := range(parts) {
					if i % 3 == 0 && part != "" {
						game.Players = append(game.Players, part)
					}
				}

				if _, ok := games[id]; ok {
					if !games[id].InProgress && game.InProgress {
						msg = fmt.Sprintf("Game started [%s : %s : %d/%d] (%s) :palm_tree: <@&%s>", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal, game.PlayerNames(), subscriberRole)
						if production {
							d.ChannelMessageSend(defaultChannelID, msg)
						}
						log.Print(msg)
					}
				} else {
					if game.InProgress {
						msg = fmt.Sprintf("Game in progress [%s : %s : %d/%d] (%s) :smiley_cat: <@&%s>", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal, game.PlayerNames(), subscriberRole)
						if production {
							d.ChannelMessageSend(defaultChannelID, msg)
						}
						log.Print(msg)
					} else {
						msg = fmt.Sprintf("New game [%s : %s : %d/%d] (%s) :smiley_cat: <@&%s>", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal, game.PlayerNames(), subscriberRole)
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
						msg = fmt.Sprintf("Game over [%s : %s : %d/%d] (%s) :fire: <@&%s>", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal, game.PlayerNames(), subscriberRole)
					} else {
						msg = fmt.Sprintf("Lobby ended [%s : %s : %d/%d] (%s) :dash: <@&%s>", game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal, game.PlayerNames(), subscriberRole)
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
		switch {
		case strings.HasPrefix(m.Content, ".games"):
			for _, game := range(games) {
				var format string
				if game.InProgress {
					format = "Game [%s : %s : %d/%d] (%s) is in progress"
				} else {
					format = "Game [%s : %s : %d/%d] (%s) is in the lobby"
				}
				msg := fmt.Sprintf(format, game.Name, game.Creator, game.SlotsTaken, game.SlotsTotal, game.PlayerNames())
				if production {
					d.ChannelMessageSend(m.ChannelID, msg)
				}
				log.Print(m.ChannelID, " ", msg)
			}
			if len(games) == 0 {
				if production {
					s.ChannelMessageSend(m.ChannelID, "No games available :crying_cat_face:")
				}
				log.Print(m.ChannelID, " ", "No games available :crying_cat_face:")
			}
		case strings.HasPrefix(m.Content, ".subscribe"):
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				log.Print(err)
				break
			}
			err = s.GuildMemberRoleAdd(channel.GuildID, m.Author.ID, subscriberRole)
			if err != nil {
				log.Print(err)
				break
			}
			dm, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				log.Print(err)
				break
			}
			s.ChannelMessageSend(dm.ID, "Successfully subscribed! :smile:")
			log.Print(m.Author.ID, " subscribed")
		case strings.HasPrefix(m.Content, ".unsubscribe"):
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				log.Print(err)
				break
			}
			err = s.GuildMemberRoleRemove(channel.GuildID, m.Author.ID, subscriberRole)
			if err != nil {
				log.Print(err)
				break
			}
			dm, err := s.UserChannelCreate(m.Author.ID)
			if err != nil {
				log.Print(err)
				break
			}
			s.ChannelMessageSend(dm.ID, "Unsubscribed :cry:")
			log.Print(m.Author.ID, " unsubscribed")
		case strings.HasPrefix(m.Content, ".stats"):
			var name string
			args := strings.SplitN(m.Content, " ", 3)
			if len(args) >= 2 {
				name = args[1]
			} else {
				name = m.Author.Username
			}

			var server, category string
			var wins, losses, games int
			var score float64
			query := fmt.Sprintf("SELECT server, wins, losses, games, score, category FROM w3mmd_elo_scores WHERE name=? AND category LIKE '%d%%' AND category LIKE '%%_league'", time.Now().Year())
			rows, err := db.Query(query, name)
			if err != nil {
				panic(err)
			}
			defer rows.Close()

			hasRow := false
			for rows.Next() {
				err := rows.Scan(&server, &wins, &losses, &games, &score, &category)
				if err != nil {
					panic(err)
				}
				hasRow = true

				percent := 100.0 * float64(wins) / float64(games)
				msg := fmt.Sprintf("%s@%s in %s: ELO(%.2f) W/L(%d/%d, %.2f%%)", name, server, category, score, wins, losses, percent)
				s.ChannelMessageSend(m.ChannelID, msg)
				log.Print(m.Author.ID, " ", msg)
			}

			if !hasRow {
				s.ChannelMessageSend(m.ChannelID, "Player not found!")
				log.Print(m.Author.ID, " Player not found!")
			}
		}
	})

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
