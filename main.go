package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/hpcloud/tail"
)

// Variables used for command line parameters
var (
	Token   string
	Logfile string
	GuildID string
)

func init() {

	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Logfile, "l", "", "Factorio Logfile")
	flag.StringVar(&GuildID, "g", "", "Discord guild ID")
	flag.Parse()
}

func parseLogLine(line *tail.Line) (string, error) {
	if line.Err != nil {
		fmt.Println("err in line: ", line.Err)
		return "", line.Err
	}
	// fmt.Println("line: ", line.Text)

	if strings.Contains(line.Text, "[CHAT]") {
		factorioChatMessage := strings.Split(line.Text, "[CHAT]")[1]
		return factorioChatMessage, nil
	}
	return "", nil
}

func logTail(tailing <-chan interface{}, c chan *tail.Line, s *discordgo.Session, ChannelID string) {
	for {
		select {
		case <-tailing:
			return

		case line := <-c:
			chatLine, err := parseLogLine(line)
			if err != nil {
				continue
			}
			if chatLine != "" {
				s.ChannelMessageSend(ChannelID, "[Factorio]"+chatLine)
			}
		}
	}
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	log.Println("Starting logfile tail")

	var talkChannel discordgo.Channel

	channels, err := dg.GuildChannels(GuildID)
	for _, channel := range channels {
		if channel.Name == "general" {
			talkChannel = *channel
		}
	}

	tailing := make(chan interface{})

	t, err := tail.TailFile(Logfile, tail.Config{Follow: true, Location: &tail.SeekInfo{Offset: 0}})
	go logTail(tailing, t.Lines, dg, talkChannel.ID)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	close(tailing)

	// cleanly exit from tail
	log.Println(t.Tell())
	t.Stop()
	t.Cleanup()

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}
}
