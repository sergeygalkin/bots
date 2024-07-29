package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/tucnak/telebot.v2"
	"gopkg.in/yaml.v3"
)

const MaxResults = 10

var bot *telebot.Bot

const CONFIG_PATH = "./config.yaml"
const MAX_COUNT_PER_FILE = "200"


type Config struct {
	BotToken string   `yaml:"bot_token"`
	BotUsers []string `yaml:"bot_users"`
	BotOwner int64    `yaml:"bot_owner"`
	DbPath   string   `yaml:"db_path"`
	FileType string   `yaml:"file_type"`
}

type SearchResult struct {
	Matches    map[string][]string
	TotalLines int // Example of an additional result
}

// GetFilesList retrieves a list of all files in the specified directory and its subdirectories.
func GetFilesList(dir string, config Config) ([]string, error) {
	var myFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == "."  + config.FileType {
			myFiles = append(myFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return myFiles, nil
}

func handleSignals(stop chan os.Signal, config Config) {
	<-stop
	if bot != nil {
		bot.Send(&telebot.User{ID: config.BotOwner}, "☠️ Exiting")
	}
	os.Exit(0)
}

func searchInFile(filePath string, keywords []string, maxLines int, config Config) SearchResult {
	results := SearchResult{
		Matches:    make(map[string][]string),
		TotalLines: 0,
	}

	if len(keywords) == 0 {
		return results
	}

	// Prepare command arguments
	args := []string{"--with-filename", "--no-heading", "--ignore-case",
		"--max-count", MAX_COUNT_PER_FILE,
		"--type", config.FileType,
		keywords[0]}
	args = append(args, filePath)

	// Execute rg command
	log.Printf("Executing rg: %v", args)
	cmd := exec.Command("rg", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error executing rg:", err)
		return results
	}

	// Parse output
	lines := strings.Split(out.String(), "\n")
	results.TotalLines = len(lines)
	count := 0

	for _, line := range lines {
		if line == "" {
			continue
		}
		// Check if the line contains all additional keywords
		if len(keywords) > 1 {
			matchesAll := true
			for _, keyword := range keywords[1:] {
				if !strings.Contains(strings.ToLower(strings.Replace(line, " ", "", -1)),
					strings.ToLower(strings.Replace(keyword, " ", "", -1))) {
					matchesAll = false
					break
				}
			}
			if !matchesAll {
				continue
			}
		}

		// Extract path and matched string
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		path := parts[0]
		matchedString := parts[1]

		// Add to results
		results.Matches[path] = append(results.Matches[path], matchedString)
		count++
		if count >= maxLines {
			break
		}
	}
	return results
}

func startBot(config Config) {
	var err error
	var lastSender telebot.Recipient
	commands := []telebot.Command{
		{Text: "search", Description: "Search for a specific item or information. Support AND as 'one|two|free'"},
		{Text: "limit", Description: "Limit output lines"},
	}
	bot, err = telebot.NewBot(telebot.Settings{
		Token:  config.BotToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalf("Failed to create bot: %s", err)
	}
	err = bot.SetCommands(commands)
	if err != nil {
		log.Fatalf("Failed to set commands: %s", err)
	}
	bot.Send(&telebot.User{ID: config.BotOwner}, "🫡 Bot started")
	myFiles, err := GetFilesList(config.DbPath, config)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	bot.Send(&telebot.User{ID: config.BotOwner},
		fmt.Sprintf("🫡 Found %d files", len(myFiles)))
	bot.Send(&telebot.User{ID: config.BotOwner},
		fmt.Sprintf("🫡 Search output limit is %d lines", MaxResults))
	search_limit := MaxResults

	// Limit command
	bot.Handle("/limit", func(m *telebot.Message) {
		if m.Sender.ID != config.BotOwner {
			log.Printf("User %s not authorized for limit command", m.Sender.Username)
			return
		}
		limit_request := m.Payload
		log.Printf("Get limit request: %s", limit_request)
		if len(limit_request) > 1 {
			search_limit, err = strconv.Atoi(strings.TrimSpace(limit_request))
			log.Printf("Set limit request to %d", search_limit)
		}
		bot.Send(&telebot.User{ID: config.BotOwner}, fmt.Sprintf("Search limit is %d now", search_limit))
	})
	// Search command
	bot.Handle("/search", func(m *telebot.Message) {
		// Restrict to a specific username
		log.Printf("Message from %s: '%s'", m.Sender.Username, m.Text)
		if m.Chat.Type == telebot.ChatPrivate {
			lastSender = m.Sender
			log.Printf("Private message from %s detected", m.Sender.Username)
		} else {
			lastSender = m.Chat
			log.Printf("Chat %s detected", m.Chat.Title)
		}

		if !slices.Contains(config.BotUsers, m.Sender.Username) {
			log.Printf("User not authorized")
			bot.Send(lastSender, "Sorry, you are not authorized to use this bot.")
			return
		}
		search_string := m.Payload
		if len(search_string) == 0 {
			bot.Send(lastSender, "🔎 Nothing to search 🤷‍♀️")
			return
		}
		substrings := strings.Split(search_string, "|")
		bot.Send(lastSender, "🔎 Searching...")
		startTime := time.Now()
		results := searchInFile(config.DbPath, substrings, search_limit, config)
		if len(results.Matches) > 0 {
			if results.TotalLines > search_limit {
				head_message := fmt.Sprintf("⚠️ The total %d lines with limits lines %s per file found, "+
					"output only %d results. This is not full results. Try to be more specific with 'City|Street|Number' on example.",
					results.TotalLines, MAX_COUNT_PER_FILE, search_limit)
				bot.Send(lastSender, head_message)
			}
			for path, matches := range results.Matches {
				parts := strings.Split(path, "/")
				file_name := parts[len(parts)-2]
				message := fmt.Sprintf("✅ %s\n- %v\n", file_name, strings.Join(matches, "\n- "))
				log.Println(message)
				bot.Send(lastSender, message)
				time.Sleep(2 * time.Second)
			}
		} else {
			log.Printf("🤷‍♀️ No results found.")
			bot.Send(lastSender, "🤷‍♀️ No results found.")
		}
		elapsed := time.Since(startTime)
		log.Printf("🏁 Finished. Total time: %s, Lines limit: %s", elapsed, strconv.Itoa(search_limit))
		bot.Send(lastSender, fmt.Sprintf("🏁 Finished. Total time is %s, the output limit is %d",
			elapsed, search_limit))
	})

	// Setup signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go handleSignals(stop, config)

	bot.Start()
}

func readConfig() Config {
	var config Config
	// Open YAML file
	file, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		log.Fatal(err)
	}
	// Decode YAML file to struct
	if err := yaml.Unmarshal(file, &config); err != nil {
		log.Fatal(err)
	}
	return config
}

func main() {
	config := readConfig()

	startBot(config)
}
