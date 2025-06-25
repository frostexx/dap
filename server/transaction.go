package server

import (
	"fmt"
	"pi/util"
	"time"

	"github.com/gin-gonic/gin"
)

type WithdrawRequest struct {
	SeedPhrase        string `json:"seed_phrase"`
	LockedBalanceID   string `json:"locked_balance_id"`
	WithdrawalAddress string `json:"withdrawal_address"`
	Amount            string `json:"amount"`
}

type WithdrawResponse struct {
	Time             string  `json:"time"`
	AttemptNumber    int     `json:"attempt_number"`
	RecipientAddress string  `json:"recipient_address"`
	SenderAddress    string  `json:"sender_address"`
	Amount           float64 `json:"amount"`
	Success          bool    `json:"success"`
	Message          string  `json:"message"`
}

func (s *Server) Withdraw(ctx *gin.Context) {
	var req WithdrawRequest

	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{
			"message": fmt.Sprintf("invalid request body: %v", err),
		})
		return
	}

	kp, err := util.GetKeyFromSeed(req.SeedPhrase)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{
			"message": "invalid seed phrase",
		})
		return
	}

	hash, amount, err := s.wallet.WithdrawClaimableBalance(kp, req.Amount, req.LockedBalanceID, req.WithdrawalAddress)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{
			"message": err.Error(),
		})
		return
	}

	res := WithdrawResponse{
		Time:             time.Now().Format("15:04:05"),
		RecipientAddress: req.WithdrawalAddress,
		SenderAddress:    s.wallet.GetAddress(kp),
		Amount:           amount,
		Message:          "Hash: " + hash,
		Success:          true,
	}

	ctx.JSON(200, res)
}
