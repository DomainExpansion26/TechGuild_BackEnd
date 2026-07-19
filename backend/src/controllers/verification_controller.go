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

// SubmitIdentityVerification godoc
// @Summary Submit identity verification
// @Description Submits an individual's identity verification request with government ID and selfie.
// @Tags Verification
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param full_name formData string true "Full Name"
// @Param govt_id_type formData string true "Government ID Type"
// @Param govt_id_number formData string true "Government ID Number"
// @Param profile_data formData string false "Additional verification data"
// @Param govt_id_document formData file true "Government ID Document"
// @Param selfie formData file true "Selfie"
// @Success 201 {object} dto.IdentityVerificationResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse
// @Router /verification/identity/submit [post]
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

// GetIdentityVerificationStatus godoc
// @Summary Get identity verification status
// @Description Returns the verification status of the authenticated user.
// @Tags Verification
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.IdentityVerificationStatusResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse
// @Router /verification/identity/status [get]

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

// SubmitBusinessVerification godoc
// @Summary Submit business verification
// @Description Submits a business verification request with the required business documents.
// @Tags Verification
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param business_name formData string true "Business Name"
// @Param gst_number formData string true "GST Number"
// @Param pan_number formData string true "PAN Number"
// @Param gst_certificate formData file true "GST Certificate"
// @Param pan_card formData file true "PAN Card"
// @Param authorized_representative_id formData file true "Authorized Representative ID"
// @Success 201 {object} dto.BusinessVerificationResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse
// @Router /verification/business/submit [post]
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

// ResubmitVerification godoc
// @Summary Resubmit verification
// @Description Resubmits a rejected verification request.
// @Tags Verification
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param record_id path string true "Verification Record ID"
// @Param reason formData string false "Additional comments"
// @Param govt_id_document formData file true "Updated Government ID Document"
// @Success 200 {object} dto.ResubmitVerificationResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 401 {object} swagger.ErrorResponse
// @Router /verification/resubmit/{record_id} [post]
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

// GetVerificationQueue godoc
// @Summary Get verification queue
// @Description Returns the list of pending verification requests for administrators.
// @Tags Admin Verification
// @Security BearerAuth
// @Produce json
// @Success 200 {array} dto.VerificationQueueItem
// @Failure 500 {object} swagger.ErrorResponse
// @Router /admin/verification/queue [get]
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

// ApproveVerification godoc
// @Summary Approve verification
// @Description Approves a verification request.
// @Tags Admin Verification
// @Security BearerAuth
// @Produce json
// @Param id path string true "Verification Record ID"
// @Success 200 {object} dto.AdminApproveResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Router /admin/verification/{id}/approve [post]
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

// RejectVerification godoc
// @Summary Reject verification
// @Description Rejects a verification request with a reason.
// @Tags Admin Verification
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Verification Record ID"
// @Param request body dto.AdminRejectRequest true "Rejection reason"
// @Success 200 {object} dto.AdminRejectResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Router /admin/verification/{id}/reject [post]
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
