package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
)

// ==========================
// Submit Individual Verification
// POST /verification/identity/submit
// ==========================

func SubmitIdentityVerification(c *gin.Context) {

	var req dto.IdentityVerificationRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	govtIDDocument, err := c.FormFile("govt_id_document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "government ID document is required",
		})
		return
	}

	selfie, err := c.FormFile("selfie")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "selfie is required",
		})
		return
	}

	// TODO:
	// Replace with user ID extracted from JWT middleware
	userID := c.GetString("user_id")

	verificationService := services.NewVerificationService(postgres.RedisDB)

	res, err := verificationService.SubmitIdentityVerification(
		userID,
		req,
		govtIDDocument,
		selfie,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ==========================
// Get Verification Status
// GET /verification/identity/status
// ==========================

func GetIdentityVerificationStatus(c *gin.Context) {

	userID := c.GetString("user_id")

	verificationService := services.NewVerificationService(postgres.RedisDB)

	res, err := verificationService.GetIdentityVerificationStatus(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ==========================
// Submit Business Verification
// POST /verification/business/submit
// ==========================

func SubmitBusinessVerification(c *gin.Context) {

	var req dto.BusinessVerificationRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	gstCertificate, err := c.FormFile("gst_certificate")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "gst_certificate is required",
		})
		return
	}

	panCard, err := c.FormFile("pan_card")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pan_card is required",
		})
		return
	}

	authorizedRepresentativeID, err := c.FormFile("authorized_representative_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "authorized_representative_id is required",
		})
		return
	}

	// User ID from JWT middleware
	userID := c.GetString("user_id")

	verificationService := services.NewVerificationService(postgres.RedisDB)

	res, err := verificationService.SubmitBusinessVerification(
		userID,
		req,
		gstCertificate,
		panCard,
		authorizedRepresentativeID,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ==========================
// Resubmit Verification
// POST /verification/resubmit/:record_id
// ==========================

func ResubmitVerification(c *gin.Context) {

	recordID := c.Param("record_id")
	var req dto.ResubmitVerificationRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	govtIDDocument, err := c.FormFile("govt_id_document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "government ID document is required",
		})
		return
	}

	userID := c.GetString("user_id")

	verificationService := services.NewVerificationService(postgres.RedisDB)

	res, err := verificationService.ResubmitVerification(
		userID,
		recordID,
		req,
		govtIDDocument,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ==========================
// Admin Verification Queue
// GET /admin/verification/queue
// ==========================

func GetVerificationQueue(c *gin.Context) {

	verificationService := services.NewVerificationService(postgres.RedisDB)

	queue, err := verificationService.GetVerificationQueue()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, queue)
}

// ==========================
// Approve Verification
// POST /admin/verification/:id/approve
// ==========================

func ApproveVerification(c *gin.Context) {

	recordID := c.Param("id")

	verificationService := services.NewVerificationService(postgres.RedisDB)

	err := verificationService.ApproveVerification(recordID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.AdminApproveResponse{
		Message: "Verification approved successfully",
	})
}

// ==========================
// Reject Verification
// POST /admin/verification/:id/reject
// ==========================

func RejectVerification(c *gin.Context) {

	recordID := c.Param("id")

	var req dto.AdminRejectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	verificationService := services.NewVerificationService(postgres.RedisDB)

	err := verificationService.RejectVerification(
		recordID,
		req.Reason,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.AdminRejectResponse{
		Message: "Verification rejected successfully",
	})
}
