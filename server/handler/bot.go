package handler

import (
	"h23s_15/api"
	"h23s_15/model"

	"github.com/labstack/echo/v4"
)

// bot投稿に対する通知の設定
// (PUT /words)
func (s Server) PutWords(ctx echo.Context) error {
	data := &api.WordBotSetting{}
	err := ctx.Bind(data)
	
	if err != nil {
		return ctx.JSON(400, err)
	}

	// traPIdの取得
	userId, err := getUserIdFromSession(ctx)
	if err != nil {
		// 正常でないためステータスコード 400 "Invalid Input"
		return ctx.JSON(400, err)
	}

	err = model.PutWords(data, userId)
	if err != nil {
		// 正常でないためステータスコード 400 "Invalid Input"
		return ctx.JSON(400, err)
	}
	
	return ctx.JSON(200, err)
}

// bot投稿に対する通知の一括設定
// (POST /words/bot)
func (s Server) PostWordsBot(ctx echo.Context) error {
	data := &api.Bot{}
	err := ctx.Bind(data)
	
	if err != nil {
		return ctx.JSON(400, err)
	}

	// traPIdの取得
	userId, err := getUserIdFromSession(ctx)
	if err != nil {
		// 正常でないためステータスコード 400 "Invalid Input"
		return ctx.JSON(400, err)
	}

	err = model.PostWordsBot(data, userId)
	if err != nil {
		// 正常でないためステータスコード 400 "Invalid Input"
		return ctx.JSON(400, err)
	}

	return ctx.JSON(200, err)
}
