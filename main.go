package main

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Global variables for the Telegram bot
var (
	telegramBotToken = "7728167799:AAHYfmG5ZEZgt_UJTDA4DdBpqoV9YBOvGRU" // Replace with your actual token
	games            = make(map[int64]*Game)                            // Store Game objects by chat ID
)

func main() {
	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			// Обработка обычных сообщений и команд
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			chatID := update.Message.Chat.ID
			userID := update.Message.From.ID

			switch update.Message.Command() {
			case "start":
				handleStartCommand(bot, chatID)
			case "new_game":
				handleNewGameCommand(bot, chatID, userID)
			case "stop":
				handleStopCommand(bot, chatID, userID)
			case "test":
				handleTestCommand(bot, chatID, userID)
			case "rules":
				handleRulesCommand(bot, chatID)
			}

		} else if update.CallbackQuery != nil {
			// Обработка callback_query

			callback := update.CallbackQuery

			// Обязательно отвечаем Telegram, чтобы избежать ошибки "query is too old"
			answerCallback := tgbotapi.NewCallback(callback.ID, "")
			if _, err := bot.Request(answerCallback); err != nil {
				log.Println("Failed to answer callback query:", err)
				continue // если не удалось ответить, пропускаем этот update
			}

			chatID := callback.Message.Chat.ID
			userID := callback.From.ID
			messageID := callback.Message.MessageID

			switch callback.Data {
			case "start_game":
				handleStartGameCallback(bot, chatID, userID, messageID)
			case "hit":
				handleHitCallback(bot, chatID, userID, messageID)
			case "stand":
				handleStandCallback(bot, chatID, userID, messageID)
			case "split":
				handleSplitCallback(bot, chatID, userID, messageID)
			default:
				log.Printf("Unknown callback data: %s", callback.Data)
			}
		}
	}
}

// --- Command and Callback Handlers ---
func handleRulesCommand(bot *tgbotapi.BotAPI, chatID int64) {
	rulesText := `📜 Правила игры в Блэкджек

Цель игры — набрать 21 очко или как можно ближе к этому значению, не превышая его.
Блэкджек — это 21 очко с двух карт (туз + карта достоинством 10). Такой результат — автоматическая победа.
Ценность карт:
   • 2–10 — по номиналу
   • Валет, дама, король — по 10 очков
   • Туз — 1 или 11 очков (считается как 11, пока сумма не превышает 21)
Начало игры:
   • Игрок и дилер получают по 2 карты
   • Одна из карт дилера закрыта
Ходы игрока:
   • Ещё карту — взять карту
   • Хватит — закончить ход, передать очередь дилеру
   • Сплит — доступен при двух одинаковых картах: рука делится на две, и к каждой добирается ещё по одной карте. Далее они играются отдельно
Ход дилера:
   • Дилер тянет карты, пока сумма его руки меньше 17
Исход игры:
   • Побеждает тот, у кого сумма ближе к 21
   • Перебор (больше 21) — автоматическое поражение

Удачи в игре! 🃏`

	msg := tgbotapi.NewMessage(chatID, rulesText)
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		log.Println("Error sending rules:", err)
	}
}
func handleStartCommand(bot *tgbotapi.BotAPI, chatID int64) {
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Новая игра", "start_game"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Нажми кнопку, чтобы начать новую игру")
	msg.ReplyMarkup = markup

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func handleNewGameCommand(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	if _, ok := games[chatID]; !ok {
		msg := tgbotapi.NewMessage(chatID, "Игра начинается...")
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
		newGame(bot, chatID, userID)
	} else {
		msg := tgbotapi.NewMessage(chatID, "Игра уже запущена")
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}

func handleStopCommand(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	end(bot, chatID, userID, "stop")
}

func handleTestCommand(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	player := NewPlayer(userID)
	game := NewGame([]*Player{player})
	game.Start()
	game.Players[userID].Hands[0].Cards[0] = NewCard(51)
	game.Players[userID].Hands[0].Cards[1] = NewCard(51)
	games[chatID] = game

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Еще карту", "hit"),
			tgbotapi.NewInlineKeyboardButtonData("Сплит", "split"),
			tgbotapi.NewInlineKeyboardButtonData("Хватит", "stand"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Карты дилера:  *%s  [ ? ]*\n\nВаши карты:  *%s*, сумма очков: *%d*",
		game.Dealer.CurrentHand.Show(),
		game.Players[userID].CurrentHand.Show(),
		game.Players[userID].CurrentHand.GetValue(),
	))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = markup

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func handleStartGameCallback(bot *tgbotapi.BotAPI, chatID int64, userID int64, messageID int) {
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	if _, err := bot.Send(editMarkup); err != nil {
		log.Println(err)
	}
	msg := tgbotapi.NewMessage(chatID, "Игра начинается...")
	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}

	if _, ok := games[chatID]; !ok {
		newGame(bot, chatID, userID)
	} else {
		msg := tgbotapi.NewMessage(chatID, "Игра уже запущена")
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}

func handleHitCallback(bot *tgbotapi.BotAPI, chatID int64, userID int64, messageID int) {
	emptyMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	if _, err := bot.Send(emptyMarkup); err != nil {
		log.Println(err)
	}

	hit(bot, chatID, userID)
}

func handleStandCallback(bot *tgbotapi.BotAPI, chatID int64, userID int64, messageID int) {
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	if _, err := bot.Send(editMarkup); err != nil {
		log.Println(err)
	}
	stand(bot, chatID, userID)
}

func handleSplitCallback(bot *tgbotapi.BotAPI, chatID int64, userID int64, messageID int) {
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	if _, err := bot.Send(editMarkup); err != nil {
		log.Println(err)
	}
	split(bot, chatID, userID)
}

// --- Game Logic Functions ---

func newGame(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	player := NewPlayer(userID)
	game := NewGame([]*Player{player})
	game.Start()
	games[chatID] = game

	hitButton := tgbotapi.NewInlineKeyboardButtonData("Еще карту", "hit")
	standButton := tgbotapi.NewInlineKeyboardButtonData("Хватит", "stand")
	splitButton := tgbotapi.NewInlineKeyboardButtonData("Сплит", "split")
	var markup tgbotapi.InlineKeyboardMarkup

	if player.CurrentHand.IsBlackjack() {
		end(bot, chatID, userID, "blackjack")
		return
	}

	if player.CurrentHand.CanSplit() {
		markup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(hitButton, splitButton, standButton))
	} else {
		markup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(hitButton, standButton))
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Карты дилера:  *%s  [ ? ]*\n\nВаши карты:  *%s*, сумма очков: *%d*",
		game.Dealer.CurrentHand.Cards[0].Show(),
		player.CurrentHand.Show(),
		player.CurrentHand.GetValue(),
	))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = markup

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func hit(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	game := games[chatID]
	player := game.Players[userID]
	player.CurrentHand.HitMe(game.Deck.Deal())

	var handLabel string
	if player.CurrentHand.IsSplited {
		if player.CurrentHand == player.Hands[0] {
			handLabel = "Первая рука: "
		} else {
			handLabel = "Вторая рука: "
		}
	} else {
		handLabel = "Ваши карты: "
	}

	hitButton := tgbotapi.NewInlineKeyboardButtonData("Еще карту", "hit")
	standButton := tgbotapi.NewInlineKeyboardButtonData("Хватит", "stand")
	newGameButton := tgbotapi.NewInlineKeyboardButtonData("Новая игра", "start_game")

	markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(hitButton, standButton))
	markupEnd := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(newGameButton))

	if player.CurrentHand.IsSplited && player.CurrentHand == player.Hands[0] && player.CurrentHand.IsBusted() {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Карты дилера:  *%s  [ ? ]*\n\n%s*%s*, сумма очков: *%d*\n\nУ вас перебор.",
			game.Dealer.CurrentHand.Cards[0].Show(),
			handLabel,
			player.CurrentHand.Show(),
			player.CurrentHand.GetValue(),
		))
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
		player.CurrentHand = player.Hands[1]

		msg = tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Карты дилера:  *%s  [ ? ]*\n\nВторая рука:  *%s*, сумма очков: *%d*",
			game.Dealer.CurrentHand.Cards[0].Show(),
			player.CurrentHand.Show(),
			player.CurrentHand.GetValue(),
		))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = markup
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	} else if !player.CurrentHand.IsBusted() {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Карты дилера:  *%s  [ ? ]*\n\n%s*%s*, сумма очков: *%d*",
			game.Dealer.CurrentHand.Cards[0].Show(),
			handLabel,
			player.CurrentHand.Show(),
			player.CurrentHand.GetValue(),
		))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = markup
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	} else if len(player.Hands) > 1 && player.CurrentHand == player.Hands[1] {
		if player.Hands[0].IsBusted() {
			end(bot, chatID, userID, "bust_both")
		} else {
			game.DealerTurn()
			for _, hand := range player.Hands {
				if (game.Dealer.CurrentHand.IsBusted() || hand.GetValue() > game.Dealer.CurrentHand.GetValue()) && !hand.IsBusted() {
					hand.Status = "победа!"
				} else if hand.GetValue() == game.Dealer.CurrentHand.GetValue() && !hand.IsBusted() {
					hand.Status = "ничья."
				} else {
					hand.Status = "проигрыш."
				}
			}

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
				"Карты дилера:  *%s* , сумма очков: *%d*\n\nПервая рука:  *%s*, сумма очков: *%d*, %s\n\nВторая рука:  *%s*, сумма очков: *%d*, %s\n\n",
				game.Dealer.CurrentHand.Show(),
				game.Dealer.CurrentHand.GetValue(),
				player.Hands[0].Show(),
				player.Hands[0].GetValue(),
				player.Hands[0].Status,
				player.Hands[1].Show(),
				player.Hands[1].GetValue(),
				player.Hands[1].Status,
			))
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = markupEnd
			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		}
	} else {
		end(bot, chatID, userID, "bust")
	}
}

func split(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	game := games[chatID]
	player := game.Players[userID]
	game.Split(player)

	hitButton := tgbotapi.NewInlineKeyboardButtonData("Еще карту", "hit")
	standButton := tgbotapi.NewInlineKeyboardButtonData("Хватит", "stand")

	markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(hitButton, standButton))

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Карты дилера:  *%s  [ ? ]*\n\nПервая рука:  *%s*, сумма очков: *%d*",
		game.Dealer.CurrentHand.Cards[0].Show(),
		player.CurrentHand.Show(),
		player.CurrentHand.GetValue(),
	))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = markup

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func stand(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	game := games[chatID]
	player := game.Players[userID]

	hitButton := tgbotapi.NewInlineKeyboardButtonData("Еще карту", "hit")
	standButton := tgbotapi.NewInlineKeyboardButtonData("Хватит", "stand")
	newGameButton := tgbotapi.NewInlineKeyboardButtonData("Новая игра", "start_game")

	markupPlay := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(hitButton, standButton))
	markupEnd := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(newGameButton))

	if player.CurrentHand.IsSplited && player.CurrentHand == player.Hands[0] {
		player.CurrentHand = player.Hands[1]
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"Карты дилера:  *%s  [ ? ]*\n\nВторая рука:  *%s*, сумма очков: *%d*",
			game.Dealer.CurrentHand.Cards[0].Show(),
			player.CurrentHand.Show(),
			player.CurrentHand.GetValue(),
		))
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = markupPlay
		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	} else {
		game.DealerTurn()
		if player.CurrentHand != player.Hands[0] {
			for _, hand := range player.Hands {
				if (game.Dealer.CurrentHand.IsBusted() || hand.GetValue() > game.Dealer.CurrentHand.GetValue()) && !hand.IsBusted() {
					hand.Status = "победа!"
				} else if hand.GetValue() == game.Dealer.CurrentHand.GetValue() && !hand.IsBusted() {
					hand.Status = "ничья."
				} else {
					hand.Status = "проигрыш."
				}
			}

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
				"Карты дилера:  *%s* , сумма очков: *%d*\n\nПервая рука:  *%s*, сумма очков: *%d*, %s\n\nВторая рука:  *%s*, сумма очков: *%d*, %s\n\n",
				game.Dealer.CurrentHand.Show(),
				game.Dealer.CurrentHand.GetValue(),
				player.Hands[0].Show(),
				player.Hands[0].GetValue(),
				player.Hands[0].Status,
				player.Hands[1].Show(),
				player.Hands[1].GetValue(),
				player.Hands[1].Status,
			))
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = markupEnd
			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		} else if game.Dealer.CurrentHand.IsBusted() || player.CurrentHand.GetValue() > game.Dealer.CurrentHand.GetValue() {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
				"Карты дилера:  *%s* , сумма очков: *%d*\n\nВаши карты: *%s*, сумма очков: *%d*\n\nПобеда!",
				game.Dealer.CurrentHand.Show(),
				game.Dealer.CurrentHand.GetValue(),
				player.CurrentHand.Show(),
				player.CurrentHand.GetValue(),
			))
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = markupEnd
			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		} else if player.CurrentHand.GetValue() == game.Dealer.CurrentHand.GetValue() {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
				"Карты дилера:  *%s* , сумма очков: *%d*\n\nВаши карты: *%s*, сумма очков: *%d*\n\nНичья!",
				game.Dealer.CurrentHand.Show(),
				game.Dealer.CurrentHand.GetValue(),
				player.CurrentHand.Show(),
				player.CurrentHand.GetValue(),
			))
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = markupEnd
			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		} else {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
				"Карты дилера:  *%s* , сумма очков: *%d*\n\nВаши карты: *%s*, сумма очков: *%d*\n\nВы проиграли.",
				game.Dealer.CurrentHand.Show(),
				game.Dealer.CurrentHand.GetValue(),
				player.CurrentHand.Show(),
				player.CurrentHand.GetValue(),
			))
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = markupEnd
			if _, err := bot.Send(msg); err != nil {
				log.Println(err)
			}
		}

		delete(games, chatID)
	}
}

func end(bot *tgbotapi.BotAPI, chatID int64, userID int64, cause string) {
	game := games[chatID]
	player := game.Players[userID]

	newGameButton := tgbotapi.NewInlineKeyboardButtonData("Новая игра", "start_game")
	markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(newGameButton))
	dealerLabel := fmt.Sprintf("Карты дилера:  *%s* , сумма очков: *%d*\n\n", game.Dealer.CurrentHand.Show(), game.Dealer.CurrentHand.GetValue())

	var msgText string

	switch cause {
	case "blackjack":
		msgText = dealerLabel + fmt.Sprintf("Ваши карты:  *%s*\n\nПобеда! У вас блэкджек!", player.CurrentHand.Show())
	case "bust":
		msgText = dealerLabel + fmt.Sprintf("Ваши карты:  *%s*, сумма очков: *%d*\n\nВы проиграли. У вас перебор.", player.CurrentHand.Show(), player.CurrentHand.GetValue())
	case "bust_both":
		msgText = dealerLabel + fmt.Sprintf("Первая рука:  *%s*, сумма очков: *%d*\n\nВторая рука:  *%s*, сумма очков: *%d*\n\nОбе руки проиграли. У вас перебор.",
			player.Hands[0].Show(), player.Hands[0].GetValue(), player.Hands[1].Show(), player.Hands[1].GetValue())
	case "stop":
		msgText = "Игра принудительно остановлена"
	default:
		msgText = "Возникла непредвиденная ошибка"
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = markup

	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}

	delete(games, chatID)
}
