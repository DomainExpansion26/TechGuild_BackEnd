package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
)

func Register(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Register endpoint working",
	})
}
func Login(c *gin.Context) {

	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authService := services.NewAuthService()

	res, err := authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}