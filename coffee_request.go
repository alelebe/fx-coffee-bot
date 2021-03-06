package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

//CoffeeRequest :
type CoffeeRequest struct {
	Bot
	initialMsg tgbotapi.Message
	chatID     int64

	Entry Beverages

	myRequests map[int]tgbotapi.Message
}

func initCoffeeRequest(bot Bot, message tgbotapi.Message) *CoffeeRequest {
	newCmd := &CoffeeRequest{
		Bot:        bot,
		initialMsg: message,
		chatID:     message.Chat.ID,

		myRequests: make(map[int]tgbotapi.Message),
	}

	const filePath = "./data/benugo.json"
	menu, err := loadBeverages(filePath)
	if err != nil {
		log.Fatal(err)
	}
	newCmd.Entry = menu.Entry

	all := newCmd.Entry.getAllEntries()
	log.Printf("Beverages loaded from file: %s, available items: %d", filePath, len(all))
	return newCmd
}

func chooseOneDrink(entry Beverages, parent *Drink) [][]tgbotapi.InlineKeyboardButton {

	drinks := entry.Items
	if parent != nil {
		drinks = append(drinks, Drink{
			ID:      fmt.Sprintf("%s::%s", btnBACK, parent.ID),
			Display: btnBACK,
		})
	}

	var keyboard [][]tgbotapi.InlineKeyboardButton
	numOfRows := len(drinks) / 2
	row := 0
	for idx := 0; idx < len(drinks); idx++ {
		item := drinks[idx]
		if row < numOfRows {
			nextItem := drinks[idx+1]
			keys := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(item.Display, item.ID),
				tgbotapi.NewInlineKeyboardButtonData(nextItem.Display, nextItem.ID),
			)
			idx++
			keyboard = append(keyboard, keys)
		} else {
			keys := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(item.Display, item.ID),
			)
			keyboard = append(keyboard, keys)
		}
		row++
	}
	return keyboard
}

func confirmChosenDrink(item Drink) [][]tgbotapi.InlineKeyboardButton {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(btnCONFIRM, fmt.Sprintf("%s::%s", btnCONFIRM, item.ID)),
		tgbotapi.NewInlineKeyboardButtonData(btnCANCEL, fmt.Sprintf("%s::%s", btnCANCEL, item.ID)),
	))
	return keyboard
}

func (p *CoffeeRequest) start() {

	sent, err := p.replyToMessageWithInlineKeyboard(
		p.initialMsg, p.Entry.Question,
		chooseOneDrink(p.Entry, nil),
	)
	if err == nil {
		p.myRequests[sent.MessageID] = sent
	}
	log.Printf("Coffee Request: new request with msgId: %d", sent.MessageID)
}

// func (p *CoffeeCmd) cancel() {
// 	if p.lastMessage != nil {
// 		p.Send(
// 			tgbotapi.NewDeleteMessage(
// 				p.chatID,
// 				p.lastMessage.MessageID,
// 		))
// 		p.lastMessage = nil
// 	}
// }

// func isSameDate(date1 int, date2 int) bool {
// 	y1, m1, d1 := time.Unix(int64(date1), 0).Date()
// 	y2, m2, d2 := time.Unix(int64(date2), 0).Date()
// 	return y1 == y2 && m1 == m2 && d1 == d2
// }

func (p CoffeeRequest) isReplyOnMyMessage(callback tgbotapi.CallbackQuery) bool {
	if callback.Message != nil {
		if _, ok := p.myRequests[callback.Message.MessageID]; ok {
			return true
		}
	}
	return false
}

func (p *CoffeeRequest) parseCallbackData(callback tgbotapi.CallbackQuery) (string, *Drink) {
	var button string
	var drink *Drink

	split := strings.Split(callback.Data, "::")
	switch len(split) {
	case 2:
		button = split[0]
		drink = p.Entry.getDrinkByID(split[1])
	case 1:
		drink = p.Entry.getDrinkByID(split[0])
	}
	return button, drink
}

func (p *CoffeeRequest) onCallback(callback tgbotapi.CallbackQuery) {

	button, drink := p.parseCallbackData(callback)
	if drink == nil {
		p.notifyUser(callback, somethingWentWrongStr)
		return
	}

	switch button {
	default: //no button -> the drink has been chosen :-)))
		p.nextQuestion(callback, *drink)

	case btnBACK:
		p.prevQuestion(callback, *drink)

	case btnCONFIRM:
		p.finishRequest(callback, *drink)

	case btnCANCEL:
		p.updateMessage(callback, "Your order cancelled...\nSorry to see you go...")
		p.removeInlineKeyboard(callback)
		// p.notifyUser(callback, "Request aborted")
	}
}

func (p *CoffeeRequest) nextQuestion(callback tgbotapi.CallbackQuery, drink Drink) {

	log.Printf("Selected: %+v", drink)

	if drink.Entry.Items == nil {
		//confirm chosen drink
		p.updateMessage(callback, fmt.Sprintf("Please confirm your choice:\n*%s*\n", drink.ID))
		p.updateInlineKeyboard(callback, confirmChosenDrink(drink))

	} else {
		//next question
		p.updateMessage(callback, drink.Entry.Question)
		p.updateInlineKeyboard(callback, chooseOneDrink(drink.Entry, &drink))
	}
}

func (p *CoffeeRequest) prevQuestion(callback tgbotapi.CallbackQuery, drink Drink) {

	log.Printf("Back to: %+v", drink)

	//back to 1st question
	p.updateMessage(callback, p.Entry.Question)
	p.updateInlineKeyboard(callback, chooseOneDrink(p.Entry, nil))
}

func (p *CoffeeRequest) finishRequest(callback tgbotapi.CallbackQuery, drink Drink) {

	log.Printf("Coffee Request: drink '%s' selected by %s", drink.ID, callback.Message.From)

	p.notifyUser(callback, "Good choice, Sir!")
	p.removeInlineKeyboard(callback)

	order := CoffeeOrder{
		UserID:    p.initialMsg.From.ID,
		UserName:  p.initialMsg.From.FirstName,
		ChatID:    p.initialMsg.Chat.ID,
		Beverage:  drink.ID,
		Price:     drink.Price,
		OrderTime: time.Now(),
	}

	if placeOrder(order) {
		p.updateMessage(callback, fmt.Sprintf("Your choice:\n*%s*\t_£%.2f_", order.Beverage, order.Price))
		p.notifyAllWatchers(fmt.Sprintf("%s: \n*%s*\t_£%.2f_", order.UserName, order.Beverage, order.Price), order.UserID)

	} else {
		p.updateMessage(callback, somethingWentWrongStr)
	}
}

func (p *CoffeeRequest) notifyAllWatchers(message string, exceptUserID int) {

	for _, obj := range allCoffeeWatchers() {
		if obj.UserID == exceptUserID {
			continue
		}

		p.sendToChat(obj.ChatID, message)
	}
}
