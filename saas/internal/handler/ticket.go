package handler

import (
	"net/http"

	"github.com/Elk-123/gotix/engine"
	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	engine engine.Engine
}

func NewTicketHandler(e engine.Engine) *TicketHandler {
	return &TicketHandler{engine: e}
}

// BookRequest 定义请求参数
type BookRequest struct {
	EventID string `json:"event_id" binding:"required"`
	SeatID  int64  `json:"seat_id" binding:"required"`
	UserID  string `json:"user_id" binding:"required"`
}

// Book 抢票接口
// POST /api/book
func (h *TicketHandler) Book(c *gin.Context) {
	var req BookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔥 调用核心引擎
	success, err := h.engine.LockSeat(c.Request.Context(), req.EventID, req.SeatID, req.UserID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统繁忙"})
		return
	}

	if !success {
		c.JSON(http.StatusConflict, gin.H{"status": "fail", "msg": "手慢了，座位已被抢!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "msg": "抢票成功! 请在5分钟内支付"})
}
