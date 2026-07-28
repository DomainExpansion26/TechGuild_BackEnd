package controllers

import (
	"net/http"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	_ "techguild-backend/src/swagger"

	"github.com/gin-gonic/gin"
)
type ContractController struct {
	service *services.ContractService
}

func NewContractController() *ContractController {
	return &ContractController{
		service: services.NewContractService(),
	}
}

// to create the contract 
func (c *ContractController) CreateContract(ctx *gin.Context) {

	clientID := ctx.GetString("userID")

	var req dto.CreateContractRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res, err := c.service.CreateContract(clientID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

//to sign the contract 
func (c *ContractController) SignContract(ctx *gin.Context) {

	userID := ctx.GetString("userID")
	contractID := ctx.Param("id")

	if err := c.service.SignContract(userID, contractID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Contract signed successfully",
	})
}

//to complete the contract
func (c *ContractController) CompleteContract(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	contractID := ctx.Param("id")

	if err := c.service.CompleteContract(clientID, contractID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Contract completed successfully",
	})
}

//to cancel the contract 
func (c *ContractController) CancelContract(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	contractID := ctx.Param("id")

	if err := c.service.CancelContract(clientID, contractID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Contract cancelled successfully",
	})
}

//get contract by ID
func (c *ContractController) GetContractByID(ctx *gin.Context) {

	contractID := ctx.Param("id")

	response, err := c.service.GetContractByID(contractID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

//get my client contracts 
func (c *ContractController) GetClientContracts(ctx *gin.Context) {

	clientID := ctx.GetString("userID")

	response, err := c.service.GetClientContracts(clientID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

//get my freelancer contract 
func (c *ContractController) GetFreelancerContracts(ctx *gin.Context) {

	freelancerID := ctx.GetString("userID")

	response, err := c.service.GetFreelancerContracts(freelancerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}