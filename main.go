package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

//Vars :
type Vars struct {
	port    string
	baseURL string

	debugBot bool
	mode     string
}

//Program :
type Program struct {
	Vars
	news   *NewsBot
	coffee *CoffeeBot
}

func (p Program) isLocal() bool {
	return p.mode == "LOCAL" || p.mode == ""
}

func configureUpdates(bot Bot) tgbotapi.UpdatesChannel {
	var err error

	_, err = bot.RemoveWebhook()
	if err != nil {
		log.Printf("Fail to get UpdatesChanel for '%s': %v\n", bot.Name, err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updatesCh, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Printf("Fail to get UpdatesChanel for '%s': %v\n", bot.Name, err)
		return nil
	}
	return updatesCh
}

func (p Program) runLongPooling() {

	var newsCh tgbotapi.UpdatesChannel
	var coffeeCh tgbotapi.UpdatesChannel

	if p.news != nil {
		newsCh = configureUpdates(p.news.Bot)
	}
	if p.coffee != nil {
		coffeeCh = configureUpdates(p.coffee.Bot)
	}

	if newsCh == nil && coffeeCh == nil {
		log.Println("Nothing to do :-( -> check logs")
		return
	}

	// читаем обновления из канала (-ов)
	for {
		select {
		case update := <-newsCh:
			p.news.dispatchMessage(update, UpdateHandler(p.news))
		case update := <-coffeeCh:
			p.coffee.dispatchMessage(update, UpdateHandler(p.coffee))
		}
	}
}

func (p Program) configureHook(bot Bot, router *gin.Engine, handler UpdateHandler) bool {
	var err error

	err = bot.setupWebhook(p.baseURL, router, handler)
	if err != nil {
		log.Printf("Fail to set WebHook for '%s': %v\n", bot.Name, err)
		return false
	}
	return true
}

func (p Program) runRouter() {
	var err error
	// run webHooks on Gin router
	router := gin.New()
	router.Use(gin.Logger())

	configured := false
	if p.news != nil {
		if p.configureHook(p.news.Bot, router, UpdateHandler(p.news)) {
			configured = true
		}
	}
	if p.coffee != nil {
		if p.configureHook(p.coffee.Bot, router, UpdateHandler(p.coffee)) {
			configured = true
		}
	}

	//start Router if at least one bot was configured and ready for work
	if configured {
		err = router.Run(":" + p.port)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Nothing to do :-( -> check logs")
	}
}

func getVar(env string) string {
	value := os.Getenv(env)
	if value == "" {
		log.Printf("$%s must be set\n", env)
	}
	return value
}
func getOptVar(env string, defValue string) string {
	value := strings.ToLower(os.Getenv(env))
	if value == "" {
		value = defValue
	}
	return value
}

func initVars() Vars {
	port := getVar("PORT")
	baseURL := getVar("BASE_URL")

	debugStr := getOptVar("DEBUG_BOT", "0")

	debug := false
	if debugStr != "" && debugStr != "0" {
		debug = true
	}

	mode := strings.ToUpper(getVar("ENV"))

	return Vars{
		port:     port,
		baseURL:  baseURL,
		debugBot: debug,
		mode:     mode,
	}
}

func main() {
	p := Program{
		Vars: initVars(),
	}
	log.Println("Started in " + p.mode + " mode")

	// MemCachier
	mcClient := initMemCache()
	defer mcClient.Quit()

	// construct Telergam Bots
	p.news = initNewsBot(getVar("NEWS_TOKEN"), p.debugBot)
	p.coffee = initCoffeeBot(getVar("COFFEE_TOKEN"), p.debugBot)

	if p.isLocal() {
		p.runLongPooling()
	} else {
		p.runRouter()
	}
}

/*
	go get gopkg.in/telegram-bot-api.v4
	heroku git:remote -a fx-coffee-bot

	govendor init
	govendor fetch github.com/gin-gonic/gin
	govendor fetch github.com/go-telegram-bot-api/telegram-bot-api

	heroku plugins:install @heroku-cli/plugin-manifest
	heroku manifest:create

	https://dashboard.ngrok.com/get-started
	./ngrok http 8080
		==> update WebhookURL
*/
