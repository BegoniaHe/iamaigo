package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"iamai/terminalcolor"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Adapter interface
type Adapter interface {
	Start() error
	Stop() error
	Name() string
}

// Plugin interface
type Plugin interface {
	HandleEvent(event Event) error
	Name() string
}

// Event struct
type Event struct {
	Name    string
	Handled bool
}

type Config struct {
	Bot struct {
		Plugins    []string `json:"plugins"`
		PluginDirs []string `json:"plugin_dirs"`
		Adapters   []string `json:"adapters"`
	} `json:"bot"`
	Log struct {
		Level            string `json:"level"`
		VerboseException bool   `json:"verbose_exception"`
	} `json:"log"`
}

// Bot struct
type Bot struct {
	config     Config
	shouldExit chan struct{}
	adapters   []Adapter
	plugins    []Plugin
	mu         sync.Mutex
}

// NewBot creates a new Bot instance
func NewBot(configFile string) (*Bot, error) {
	bot := &Bot{
		config:     Config{},
		shouldExit: make(chan struct{}),
		adapters:   []Adapter{},
		plugins:    []Plugin{},
	}

	if err := bot.loadConfig(configFile); err != nil {
		return nil, err
	}

	return bot, nil
}

// Load configuration file
func (b *Bot) loadConfig(configFile string) error {
	if configFile == "" {
		return errors.New("config file not specified")
	}
	terminalcolor.LogWithColorAndStyle{
		Text:   []string{configFile},
		Color:  []string{"info", "reset"},
		Format: "Loading config file {text[0]}",
	}.Print()
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&b.config); err != nil {
		return err
	}
	return nil
}

func (b *Bot) showBotState() {
	terminalcolor.LogWithColorAndStyle{
		Text: []string{b.config.Log.Level},
		Color: func() []string {
			switch strings.ToLower(b.config.Log.Level) {
			case "info":
				return []string{"info", "green"}
			case "warning":
				return []string{"info", "yellow"}
			case "error":
				return []string{"info", "red"}
			default:
				return []string{"info"}
			}
		}(),
		Style:  []string{"", "bold"},
		Format: "Bot is running with log level {text[0]}",
	}.Print()
	terminalcolor.LogWithColorAndStyle{
		Text:   []string{fmt.Sprint(len(b.adapters))},
		Color:  []string{"info"},
		Format: "Adapters: {text[0]}",
	}.Print()
	terminalcolor.LogWithColorAndStyle{
		Text:   []string{fmt.Sprint(len(b.plugins))},
		Color:  []string{"info"},
		Format: "Plugins: {text[0]}",
	}.Print()
}

// Run the Bot
func (b *Bot) Run() {
	// Handle exit signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	b.showBotState()

	go func() {
		<-sigs
		terminalcolor.LogWithColorAndStyle{
			Text:  []string{"Received keyboard interrupt"},
			Color: []string{"warning"},
		}.Print()
		close(b.shouldExit)
	}()

	for _, adapter := range b.adapters {
		go func(a Adapter) {
			if err := a.Start(); err != nil {
				terminalcolor.LogWithColorAndStyle{
					Text:   []string{a.Name(), err.Error()},
					Color:  []string{"error"},
					Format: "Error starting adapter {text[0]}: {text[1]}",
				}.Print()
			}
		}(adapter)
	}

	for {
		select {
		case <-b.shouldExit:
			terminalcolor.LogWithColorAndStyle{
				Text:   []string{"Exiting..."},
				Style:  []string{"", "bold"},
				Color:  []string{"warning"},
				Format: "{text[0]}",
			}.Print()
			for _, adapter := range b.adapters {
				if err := adapter.Stop(); err != nil {
					terminalcolor.LogWithColorAndStyle{
						Text:   []string{adapter.Name(), err.Error()},
						Color:  []string{"error"},
						Format: "Error stopping adapter {text[0]}: {text[1]}",
					}.Print()
				}
			}
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// AddAdapter adds an adapter to the bot
func (b *Bot) AddAdapter(adapter Adapter) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.adapters = append(b.adapters, adapter)
}

// AddPlugin adds a plugin to the bot
func (b *Bot) AddPlugin(plugin Plugin) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.plugins = append(b.plugins, plugin)
}

// HandleEvent processes an event using plugins
func (b *Bot) HandleEvent(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, plugin := range b.plugins {
		if err := plugin.HandleEvent(event); err != nil {
			terminalcolor.LogWithColorAndStyle{
				Text:   []string{plugin.Name(), err.Error()},
				Color:  []string{"error"},
				Format: "Error handling event with plugin {text[0]}: {text[1]}",
			}.Print()
		}
	}
}

// ReloadPlugins reloads all plugins
func (b *Bot) ReloadPlugins() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.plugins = []Plugin{}
	// Add logic to reload plugins
}

// GetAdapter retrieves an adapter by name
func (b *Bot) GetAdapter(name string) (Adapter, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, adapter := range b.adapters {
		if adapter.Name() == name {
			return adapter, nil
		}
	}
	return nil, errors.New("adapter not found")
}

// GetPlugin retrieves a plugin by name
func (b *Bot) GetPlugin(name string) (Plugin, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, plugin := range b.plugins {
		if plugin.Name() == name {
			return plugin, nil
		}
	}
	return nil, errors.New("plugin not found")
}

// LoadPluginsFromDir loads plugins from a directory
func (b *Bot) LoadPluginsFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return err
	}

	for _, file := range files {
		p, err := plugin.Open(file)
		if err != nil {
			return err
		}

		symPlugin, err := p.Lookup("Plugin")
		if err != nil {
			return err
		}

		plugin, ok := symPlugin.(Plugin)
		if !ok {
			return errors.New("unexpected type from module symbol")
		}

		b.AddPlugin(plugin)
	}

	return nil
}

func main() {
	bot, err := NewBot("config.json")
	if err != nil {
		terminalcolor.LogWithColorAndStyle{
			Text:  []string{err.Error()},
			Color: []string{"error"},
		}.Print()
		return
	}

	// Load plugins
	err = bot.LoadPluginsFromDir("./plugins")
	if err != nil {
		terminalcolor.LogWithColorAndStyle{
			Text:  []string{err.Error()},
			Color: []string{"error"},
		}.Print()
		return
	}

	bot.Run()
}
