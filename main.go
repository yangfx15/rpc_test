package main

import (
	"encoding/json"
	"fmt"
	"github.com/noirbizarre/gonja"
	"io"
	"log"
	"rpc_test/protos/gpt"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("/stream", HeadersMiddleware(), func(c *gin.Context) {
		ch := make(chan string, 0)
		appId := c.Query("appId")
		go func() {
			for i := 0; i < 10; i++ {
				value := fmt.Sprintf("%v:%v", i, appId)
				reply := gpt.GPTReply{Message: value}
				bts, _ := json.Marshal(reply)
				ch <- string(bts)
				time.Sleep(time.Second)
			}
		}()
		c.Stream(func(w io.Writer) bool {
			// Stream message to client from message channel
			if msg, ok := <-ch; ok {
				c.SSEvent("message", msg)
				return true
			}
			return false
		})
	})

	router.POST("/render", func(c *gin.Context) {
		tpl := gonja.Must(gonja.FromFile("templates/gpt-3_5-turbo-0301-do_chat.txt"))

		var dialogs [][]string
		dialogs = append(dialogs, []string{"123", "456"})

		out, err := tpl.Execute(gonja.Context{
			"context": dialogs,
			"query":   "789",
		})
		if err != nil {

		}

		log.Printf("out: %v\n", out)
	})

	router.Run(":8844")
}

func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Next()
	}
}
