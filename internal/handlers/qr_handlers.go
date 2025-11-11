package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"url_shortener/internal/qrcode"
	"url_shortener/internal/telemetry"

	"github.com/gin-gonic/gin"
)

// QRCodeRequest represents the request body for generating a QR code via POST
type QRCodeRequest struct {
	Data                 string  `json:"data" binding:"required" example:"https://example.com" description:"The data to encode in the QR code (required)"`
	Size                 *int    `json:"size,omitempty" example:"512" description:"Output image size in pixels (default: 256, min: 64, max: 2048)"`
	ErrorCorrection      *string `json:"error_correction,omitempty" example:"high" description:"Error correction level: low, medium, high, highest (default: high)"`
	ForegroundColor      *string `json:"foreground_color,omitempty" example:"#000000" description:"QR code foreground color in hex (default: #000000)"`
	BackgroundColor      *string `json:"background_color,omitempty" example:"#FFFFFF" description:"Background color in hex (default: #FFFFFF)"`
	TransparentBackground *bool   `json:"transparent_background,omitempty" example:"false" description:"Make background transparent (default: false)"`
	IncludeLogo          *bool   `json:"include_logo,omitempty" example:"true" description:"Include logo in center (default: true)"`
	LogoColor            *string `json:"logo_color,omitempty" example:"#FF5733" description:"Logo color in hex (optional, uses original if not set)"`
	LogoShape            *string `json:"logo_shape,omitempty" example:"circle" description:"Logo shape: circle or square (default: circle)"`
	ModuleShape          *string `json:"module_shape,omitempty" example:"square" description:"QR module shape: square, circle, rounded (default: square)"`
	BorderWidth          *int    `json:"border_width,omitempty" example:"2" description:"Border width in modules (default: 2, min: 0, max: 10)"`
	Format               *string `json:"format,omitempty" example:"png" description:"Output format: png or jpeg (default: png)"`
}

// GenerateQRCodePOST handles POST requests for QR code generation with JSON body
// @Summary Generate QR code (POST)
// @Description Generate a QR code with full customization options via JSON body
// @Tags qrcode
// @Accept json
// @Produce image/png,image/jpeg
// @Param qr body QRCodeRequest true "QR code generation request"
// @Success 200 {file} binary "QR code image"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /qr [post]
func (h *Handler) GenerateQRCodePOST(c *gin.Context) {
	_, span := telemetry.StartSpan(c.Request.Context(), "generate_qr_post")
	defer span.End()

	var req QRCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build options from request
	opts := buildQROptions(req.Data, &req)

	// Generate QR code
	imgData, err := qrcode.Generate(opts)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set content type and return image
	contentType := "image/png"
	if opts.Format == "jpeg" {
		contentType = "image/jpeg"
	}

	c.Data(http.StatusOK, contentType, imgData)
}

// GenerateQRCodeGET handles GET requests for QR code generation with query parameters
// @Summary Generate QR code (GET)
// @Description Generate a QR code with customization options via query parameters
// @Tags qrcode
// @Produce image/png,image/jpeg
// @Param data query string true "The data to encode in the QR code"
// @Param size query int false "Output image size in pixels (default: 256, min: 64, max: 2048)"
// @Param error_correction query string false "Error correction level: low, medium, high, highest (default: high)"
// @Param foreground_color query string false "QR code foreground color in hex (default: #000000)"
// @Param background_color query string false "Background color in hex (default: #FFFFFF)"
// @Param transparent_background query bool false "Make background transparent (default: false)"
// @Param include_logo query bool false "Include logo in center (default: true)"
// @Param logo_color query string false "Logo color in hex (optional)"
// @Param logo_shape query string false "Logo shape: circle or square (default: circle)"
// @Param module_shape query string false "QR module shape: square, circle, rounded (default: square)"
// @Param border_width query int false "Border width in modules (default: 2, min: 0, max: 10)"
// @Param format query string false "Output format: png or jpeg (default: png)"
// @Success 200 {file} binary "QR code image"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /qr [get]
func (h *Handler) GenerateQRCodeGET(c *gin.Context) {
	_, span := telemetry.StartSpan(c.Request.Context(), "generate_qr_get")
	defer span.End()

	// Get required data parameter
	data := c.Query("data")
	if data == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data query parameter is required"})
		return
	}

	// Parse optional parameters
	var req QRCodeRequest
	req.Data = data

	// Parse size
	if sizeStr := c.Query("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			req.Size = &size
		}
	}

	// Parse error correction
	if ec := c.Query("error_correction"); ec != "" {
		req.ErrorCorrection = &ec
	}

	// Parse foreground color
	if fg := c.Query("foreground_color"); fg != "" {
		req.ForegroundColor = &fg
	}

	// Parse background color
	if bg := c.Query("background_color"); bg != "" {
		req.BackgroundColor = &bg
	}

	// Parse transparent background
	if tb := c.Query("transparent_background"); tb != "" {
		if val, err := strconv.ParseBool(tb); err == nil {
			req.TransparentBackground = &val
		}
	}

	// Parse include logo
	if il := c.Query("include_logo"); il != "" {
		if val, err := strconv.ParseBool(il); err == nil {
			req.IncludeLogo = &val
		}
	}

	// Parse logo color
	if lc := c.Query("logo_color"); lc != "" {
		req.LogoColor = &lc
	}

	// Parse logo shape
	if ls := c.Query("logo_shape"); ls != "" {
		req.LogoShape = &ls
	}

	// Parse module shape
	if ms := c.Query("module_shape"); ms != "" {
		req.ModuleShape = &ms
	}

	// Parse border width
	if bw := c.Query("border_width"); bw != "" {
		if val, err := strconv.Atoi(bw); err == nil {
			req.BorderWidth = &val
		}
	}

	// Parse format
	if fmt := c.Query("format"); fmt != "" {
		req.Format = &fmt
	}

	// Build options from request
	opts := buildQROptions(data, &req)

	// Generate QR code
	imgData, err := qrcode.Generate(opts)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set content type and return image
	contentType := "image/png"
	if opts.Format == "jpeg" {
		contentType = "image/jpeg"
	}

	c.Data(http.StatusOK, contentType, imgData)
}

// buildQROptions builds QR code options from request parameters with defaults
func buildQROptions(data string, req *QRCodeRequest) qrcode.Options {
	opts := qrcode.DefaultOptions()
	opts.Data = data

	// Apply custom values if provided
	if req.Size != nil {
		opts.Size = *req.Size
	}

	if req.ErrorCorrection != nil {
		opts.ErrorCorrection = strings.ToLower(*req.ErrorCorrection)
	}

	if req.ForegroundColor != nil {
		opts.ForegroundColor = *req.ForegroundColor
	}

	if req.BackgroundColor != nil {
		opts.BackgroundColor = *req.BackgroundColor
	}

	if req.TransparentBackground != nil {
		opts.TransparentBackground = *req.TransparentBackground
	}

	if req.IncludeLogo != nil {
		opts.IncludeLogo = *req.IncludeLogo
	}

	if req.LogoColor != nil {
		opts.LogoColor = *req.LogoColor
	}

	if req.LogoShape != nil {
		opts.LogoShape = strings.ToLower(*req.LogoShape)
	}

	if req.ModuleShape != nil {
		opts.ModuleShape = strings.ToLower(*req.ModuleShape)
	}

	if req.BorderWidth != nil {
		opts.BorderWidth = *req.BorderWidth
	}

	if req.Format != nil {
		opts.Format = strings.ToLower(*req.Format)
	}

	return opts
}
