package main

import (
	"flag"
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

var (
	pluginDir = flag.String("plugindir", "bin", "Default dir to search for plugins")
	debugLog  = flag.Bool("debug", false, "Debug messages and faster config saving")
)

func main() {
	flag.Parse()
	log.Printf("Running on Go version: %s\n", runtime.Version())

	// Load configs before anything else, as it will be needed
	bot.LoadConfig()
	bot.LoadPluginConfig()
	var token = bot.C.BotToken
	if token == "" {
		log.Fatalln("No bot_token given")
	}

	s := state.NewWithIntents("Bot "+token,
		gateway.IntentGuildMessages,
		gateway.IntentGuildEmojis,
		gateway.IntentGuildMessageReactions,
		gateway.IntentDirectMessages,
		gateway.IntentGuildMembers,
	)
	bot.Client = *s

	if s == nil {
		log.Fatalln("Session failed: is nil")
	}

	// Add handlers
	s.AddHandler(func(e *gateway.MessageCreateEvent) {
		go cmd.CommandHandler(e)
		go cmd.ResponseHandler(e)
	})
	s.AddHandler(func(e *gateway.GuildMemberUpdateEvent) {
		go cmd.UpdateMemberCache(e)
	})

	if err := s.Open(bot.Ctx); err != nil {
		log.Fatalln("Failed to connect:", err)
	}

	// Cancel context when SIGINT / SIGKILL / SIGTERM. SIGTERM is used by `docker stop`
	ctx, cancel := signal.NotifyContext(bot.Ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	if err := s.Open(ctx); err != nil {
		log.Println("cannot open:", err)
	}

	u, err := s.Me()
	if err != nil {
		log.Fatalln("Failed to get bot user:", err)
	}
	bot.User = u

	// We want http bash requests immediately accessible just in case something needs them.
	// Though, this shouldn't really ever happen, it doesn't hurt.
	util.RegisterHttpBashRequests()

	// Call plugins after logging in with the bot, but before doing anything else at all
	go plugins.RegisterAll(*pluginDir)

	// Set up the bots status
	go bot.LoadActivityStatus()

	// Now we can start the routine-based tasks
	go bot.SetupConfigSaving()
	go bot.Scheduler.StartAsync()

	log.Printf("Started as %v (%s#%s). Debugging is set to `%v`.\n", u.ID, u.Username, u.Discriminator, *debugLog)

	go checkGuildCounts(s)

	<-ctx.Done() // block until Ctrl+C / SIGINT / SIGTERM

	log.Println("received signal, shutting down")

	bot.SaveConfig()
	bot.SavePluginConfig()
	plugins.SaveConfig()
	plugins.Shutdown()

	if err := s.Close(); err != nil {
		log.Println("cannot close:", err)
	}

	log.Println("closed connection")
}

func checkGuildCounts(s *state.State) {
	guilds, err := s.Guilds()
	if err != nil {
		log.Printf("checkGuildCounts: %v\n", err)
	}

	fmtGuilds := make([]string, 0)
	members := 0
	for _, guild := range guilds {
		if guildMembers, err := s.Members(guild.ID); err == nil {
			numMembers := len(guildMembers)
			members += numMembers
			fmtGuilds = append(fmtGuilds, fmt.Sprintf("- %v - %s - (%s)", guild.ID, guild.Name, util.JoinIntAndStr(numMembers, "member")))
		}
	}

	log.Printf(
		"Currently serving %s on %s\n%s",
		util.JoinIntAndStr(members, "user"),
		util.JoinIntAndStr(len(guilds), "guild"),
		strings.Join(fmtGuilds, "\n"),
	)
}
