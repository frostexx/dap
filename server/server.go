package server

import (
	"fmt"
	"pi/wallet"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	wallet *wallet.Wallet
}

func New() *Server {
	return &Server{
		wallet: wallet.New(),
	}
}

func (s *Server) Run(port string) error {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // or restrict to specific domains
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Static("/", "./public")
	r.POST("/api/login", s.Login)
	r.POST("/api/withdraw", s.Withdraw)

	fmt.Printf("running on port: %s\n", port)

	r.Run(port)
	return nil
}
