package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/verbeux-ai/whatsmiau/lib/whatsmiau"
	"github.com/verbeux-ai/whatsmiau/server/controllers"
)

func Status(group *echo.Group) {
	controller := controllers.NewStatus(whatsmiau.Get())

	group.POST("/text", controller.SendText)
	group.POST("/image", controller.SendImage)
	group.POST("/video", controller.SendVideo)
	group.POST("/audio", controller.SendAudio)
}
