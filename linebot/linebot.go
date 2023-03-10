package linebot

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/hcyang1106/awesomeProject/config"
	"github.com/hcyang1106/awesomeProject/model"
	"github.com/hcyang1106/awesomeProject/repository"
	linebot2 "github.com/line/line-bot-sdk-go/v7/linebot"
	"log"
	"net/http"
)

type LineBot struct {
	Router *gin.Engine
	Repo   *repository.Repository
	Bot    *linebot2.Client
	Config *config.Config
}

func NewLineBot() *LineBot {
	linebot := &LineBot{}
	router := gin.Default()
	config := config.NewConfig()
	repo := repository.NewRepository(config)
	bot, err := linebot2.New(
		config.ChannelSecret,
		config.ChannelAccessToken,
	)
	if err != nil {
		log.Print(err)
	}
	linebot.Router = router
	linebot.Repo = repo
	linebot.Config = config
	linebot.Bot = bot
	return linebot
}

func (l *LineBot) Start() {
	l.Router.GET("/history", l.GetHistoriesByName)
	l.Router.POST("/message", l.SendMessageToName)
	l.Router.POST("/history", l.CreateHistory)
	l.Router.Run(l.Config.Address)
}

func (l *LineBot) GetHistoriesByName(c *gin.Context) {
	name := c.Query("name")
	histories, err := l.Repo.GetHistoriesByName(name)
	if err != nil {
		log.Print(err)
	}
	json, err := json.Marshal(histories)
	if err != nil {
		log.Print(err)
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8;", json)
}

func (l *LineBot) SendMessageToName(c *gin.Context) {
	name := c.PostForm("name")
	msg := c.PostForm("msg")

	history, err := l.Repo.FindOneHistoryByName(name)
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusOK, gin.H{"status": "db error"})
		return
	}
	if history == nil {
		c.JSON(http.StatusOK, gin.H{"status": "uid not found"})
		return
	}

	_, err = l.Bot.PushMessage(history.Uid, linebot2.NewTextMessage(msg)).Do()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "send message failed"})
		log.Print(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "message sent"})
}

func (l *LineBot) CreateHistory(c *gin.Context) {
	events, err := l.Bot.ParseRequest(c.Request)
	if err != nil {
		if err == linebot2.ErrInvalidSignature {
			c.JSON(http.StatusBadRequest, gin.H{})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{})
		}
		return
	}
	for _, event := range events {
		profile, err := l.Bot.GetProfile(event.Source.UserID).Do()
		if err != nil {
			log.Print(err)
		}
		if event.Type == linebot2.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot2.TextMessage:
				history := &model.History{
					Name:    profile.DisplayName,
					Message: message.Text,
					Uid:     event.Source.UserID,
				}
				if err := l.Repo.CreateHistory(history); err != nil {
					log.Print(err)
				}
				if _, err = l.Bot.ReplyMessage(event.ReplyToken, linebot2.NewTextMessage("message saved")).Do(); err != nil {
					log.Print(err)
				}
			}
		}
	}
}
