package main

import (
	"fmt"
	"github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/ryo-chin/go-web-frameworks/internal/gin/auth"
	"github.com/ryo-chin/go-web-frameworks/internal/gin/chat"
	"io"
	"log"
	"math/rand"
	"net/http"
)

var roomManager *chat.Manager

func main() {
	roomManager = chat.NewRoomManager()
	router := gin.Default()
	router.SetHTMLTemplate(chat.Html)

	authMiddleWare, err := auth.NewJWTMiddleWare()
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}
	router.NoRoute(authMiddleWare.MiddlewareFunc(), func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		log.Printf("NoRoute claims: %#v\n", claims)
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	// Auth
	router.POST("/login/jwt", authMiddleWare.LoginHandler)
	authRouter := router.Group("/auth")
	authRouter.GET("/refresh_token", authMiddleWare.RefreshHandler) // Refresh time can be longer than token timeout
	authRouter.Use(authMiddleWare.MiddlewareFunc())
	{
		authRouter.GET("/hello", func(c *gin.Context) {
			claims := jwt.ExtractClaims(c)
			user, _ := c.Get(auth.IdentityKey)
			c.JSON(200, gin.H{
				"userID":   claims[auth.IdentityKey],
				"userName": user.(*auth.User).UserName,
				"text":     "Hello World.",
			})
		})
	}

	// Chat
	router.GET("/room/:roomid", roomGET)
	router.POST("/room/:roomid", roomPOST)
	router.DELETE("/room/:roomid", roomDELETE)
	router.GET("/stream/:roomid", stream)

	router.Run(":8080")
}

func stream(c *gin.Context) {
	roomid := c.Param("roomid")
	listener := roomManager.OpenListener(roomid)
	defer roomManager.CloseListener(roomid, listener)

	clientGone := c.Writer.CloseNotify()
	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case message := <-listener:
			c.SSEvent("message", message)
			return true
		}
	})
}

func roomGET(c *gin.Context) {
	roomid := c.Param("roomid")
	userid := fmt.Sprint(rand.Int31())
	c.HTML(http.StatusOK, "chat_room", gin.H{
		"roomid": roomid,
		"userid": userid,
	})
}

func roomPOST(c *gin.Context) {
	roomid := c.Param("roomid")
	userid := c.PostForm("user")
	message := c.PostForm("message")
	roomManager.Submit(userid, roomid, message)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
	})
}

func roomDELETE(c *gin.Context) {
	roomid := c.Param("roomid")
	roomManager.DeleteBroadcast(roomid)
}
