package qrcode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// Options represents configuration options for QR code generation
type Options struct {
	Data                 string
	Size                 int
	ErrorCorrection      string
	ForegroundColor      string
	BackgroundColor      string
	TransparentBackground bool
	IncludeLogo          bool
	LogoColor            string
	LogoShape            string
	ModuleShape          string
	BorderWidth          int
	Format               string
}

// DefaultOptions returns default QR code generation options
func DefaultOptions() Options {
	return Options{
		Size:                 256,
		ErrorCorrection:      "high",
		ForegroundColor:      "#000000",
		BackgroundColor:      "#FFFFFF",
		TransparentBackground: false,
		IncludeLogo:          true,
		LogoColor:            "",
		LogoShape:            "circle",
		ModuleShape:          "square",
		BorderWidth:          2,
		Format:               "png",
	}
}

// Generate creates a QR code with the given options and returns the image bytes
func Generate(opts Options) ([]byte, error) {
	// Use new implementation with skip2/go-qrcode
	return GenerateWithSkip(opts)
}

// GenerateOld is the old yeqown implementation (kept for reference)
func GenerateOld(opts Options) ([]byte, error) {
	// Validate required fields
	if opts.Data == "" {
		return nil, fmt.Errorf("data is required")
	}

	// Validate size
	if opts.Size < 64 || opts.Size > 2048 {
		return nil, fmt.Errorf("size must be between 64 and 2048")
	}

	// Validate color formats
	if err := validateHexColor(opts.ForegroundColor); err != nil {
		return nil, fmt.Errorf("invalid foreground_color: %w", err)
	}
	if err := validateHexColor(opts.BackgroundColor); err != nil {
		return nil, fmt.Errorf("invalid background_color: %w", err)
	}
	if opts.LogoColor != "" {
		if err := validateHexColor(opts.LogoColor); err != nil {
			return nil, fmt.Errorf("invalid logo_color: %w", err)
		}
	}

	// Create QR code (error correction is set via WithRecoveryLevel on writer)
	qrc, err := qrcode.New(opts.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	// Create a temporary file for output
	tmpFile, err := os.CreateTemp("", "qr-*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Prepare writer options
	writerOpts := []standard.ImageOption{
		standard.WithBgColorRGBHex(opts.BackgroundColor),
		standard.WithFgColorRGBHex(opts.ForegroundColor),
		standard.WithBorderWidth(opts.BorderWidth),
	}

	// Add module shape
	switch opts.ModuleShape {
	case "circle":
		writerOpts = append(writerOpts, standard.WithCircleShape())
	case "rounded":
		// Note: rounded rectangles might not be supported in all versions
		// Fallback to square if not available
		writerOpts = append(writerOpts, standard.WithCircleShape())
	}

	// Calculate QR width based on size
	qrWidth := calculateQRWidth(opts.Size)
	writerOpts = append(writerOpts, standard.WithQRWidth(uint8(qrWidth)))

	// Force PNG output format if transparency is requested (JPEG doesn't support transparency)
	if opts.TransparentBackground {
		writerOpts = append(writerOpts, standard.WithBuiltinImageEncoder(standard.PNG_FORMAT))
	}

	// Add logo if requested
	if opts.IncludeLogo {
		logoPath := "internal/assets/logo.png"

		// Check if logo exists
		if _, err := os.Stat(logoPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("logo file not found at %s", logoPath)
		}

		// Resize logo - 1/4 of QR code for good visibility
		// The library's safe zone will handle creating blank space in QR data
		logoSize := opts.Size / 4
		resizedLogoPath, err := resizeLogo(logoPath, logoSize)
		if err != nil {
			return nil, fmt.Errorf("failed to resize logo: %w", err)
		}
		defer os.Remove(resizedLogoPath)

		// If logo color is specified, recolor the logo
		if opts.LogoColor != "" {
			recoloredLogoPath, err := recolorLogo(resizedLogoPath, opts.LogoColor)
			if err != nil {
				return nil, fmt.Errorf("failed to recolor logo: %w", err)
			}
			defer os.Remove(recoloredLogoPath)
			resizedLogoPath = recoloredLogoPath
		}

		// Load logo and let the library handle the safe zone
		// WithLogoSafeZone() tells the QR encoder to avoid placing data in the logo area
		writerOpts = append(writerOpts,
			standard.WithLogoImageFilePNG(resizedLogoPath),
			standard.WithLogoSizeMultiplier(3), // Make logo area 3x larger for proper safe zone
			standard.WithBgTransparent(),       // Use transparent background for logo area
		)
	}

	// Create writer
	w, err := standard.New(tmpFile.Name(), writerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	// Save QR code
	if err := qrc.Save(w); err != nil {
		return nil, fmt.Errorf("failed to save QR code: %w", err)
	}

	// Read the generated file
	imgData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read generated QR code: %w", err)
	}

	// Handle transparent background if requested
	if opts.TransparentBackground {
		imgData, err = makeBackgroundTransparent(imgData, opts.BackgroundColor)
		if err != nil {
			return nil, fmt.Errorf("failed to make background transparent: %w", err)
		}
	}

	return imgData, nil
}

// Note: Error correction level is not configurable in yeqown/go-qrcode/v2
// The library uses optimal error correction based on data size automatically

// calculateQRWidth determines the appropriate QR module width based on output size
func calculateQRWidth(size int) int {
	// QR codes have 21-177 modules depending on version
	// We'll use a sensible default based on size
	if size <= 128 {
		return 21 // Version 1
	} else if size <= 256 {
		return 25 // Version 2
	} else if size <= 512 {
		return 29 // Version 3
	} else {
		return 33 // Version 4
	}
}

// addLogoSafeZone adds a border around the logo with the specified background color
func addLogoSafeZone(logoPath, bgColorHex string) (string, error) {
	// Parse background color
	bgColor, err := parseHexColor(bgColorHex)
	if err != nil {
		return "", err
	}

	// Open the logo
	file, err := os.Open(logoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open logo: %w", err)
	}
	defer file.Close()

	// Decode the logo
	logo, err := png.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode logo: %w", err)
	}

	bounds := logo.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Add 20% padding on each side as safe zone
	padding := width / 5
	newWidth := width + (padding * 2)
	newHeight := height + (padding * 2)

	// Create new image with safe zone
	withSafeZone := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Fill with background color
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			withSafeZone.Set(x, y, bgColor)
		}
	}

	// Draw original logo in center
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			withSafeZone.Set(x+padding, y+padding, logo.At(x, y))
		}
	}

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "logo-safezone-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, withSafeZone); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to encode logo with safe zone: %w", err)
	}

	return tmpFile.Name(), nil
}

// resizeLogo resizes a logo to fit within maxSize while maintaining aspect ratio
func resizeLogo(logoPath string, maxSize int) (string, error) {
	// Open the original logo
	file, err := os.Open(logoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open logo: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, err := png.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode logo: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		newWidth = maxSize
		newHeight = (height * maxSize) / width
	} else {
		newHeight = maxSize
		newWidth = (width * maxSize) / height
	}

	// Ensure minimum size
	if newWidth < 10 {
		newWidth = 10
	}
	if newHeight < 10 {
		newHeight = 10
	}

	// Create new image with resized dimensions
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple nearest-neighbor resizing
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := (x * width) / newWidth
			srcY := (y * height) / newHeight
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}

	// Save resized logo to temp file
	tmpFile, err := os.CreateTemp("", "logo-resized-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, resized); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to encode resized logo: %w", err)
	}

	return tmpFile.Name(), nil
}

// recolorLogo recolors a PNG logo to the specified color while preserving transparency
func recolorLogo(logoPath, hexColor string) (string, error) {
	// Parse the target color
	targetColor, err := parseHexColor(hexColor)
	if err != nil {
		return "", err
	}

	// Open the original logo
	file, err := os.Open(logoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open logo: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, err := png.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode logo: %w", err)
	}

	// Create a new RGBA image
	bounds := img.Bounds()
	recolored := image.NewRGBA(bounds)

	// Recolor each pixel
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldColor := img.At(x, y)
			_, _, _, a := oldColor.RGBA()

			// If pixel is not fully transparent, replace with target color
			// but preserve alpha channel
			if a > 0 {
				newColor := color.RGBA{
					R: targetColor.R,
					G: targetColor.G,
					B: targetColor.B,
					A: uint8(a >> 8), // Convert from 16-bit to 8-bit
				}
				recolored.Set(x, y, newColor)
			} else {
				// Preserve transparency
				recolored.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			}
		}
	}

	// Save recolored logo to temp file
	tmpFile, err := os.CreateTemp("", "logo-recolored-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, recolored); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to encode recolored logo: %w", err)
	}

	return tmpFile.Name(), nil
}

// makeBackgroundTransparent makes the background of a QR code transparent
func makeBackgroundTransparent(imgData []byte, bgColorHex string) ([]byte, error) {
	// Parse background color to match
	bgColor, err := parseHexColor(bgColorHex)
	if err != nil {
		return nil, err
	}

	// Decode the image
	img, err := png.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Create a new RGBA image
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	// Make background pixels transparent
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldColor := img.At(x, y)
			r, g, b, a := oldColor.RGBA()

			// Convert to 8-bit
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			// Check if color matches background (with some tolerance)
			if colorMatches(r8, g8, b8, bgColor, 10) {
				// Make transparent
				rgba.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			} else {
				// Keep original color
				rgba.Set(x, y, color.RGBA{R: r8, G: g8, B: b8, A: uint8(a >> 8)})
			}
		}
	}

	// Encode back to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("failed to encode transparent image: %w", err)
	}

	return buf.Bytes(), nil
}

// colorMatches checks if two colors match within a tolerance
func colorMatches(r1, g1, b1 uint8, c2 color.RGBA, tolerance uint8) bool {
	return abs(int(r1)-int(c2.R)) <= int(tolerance) &&
		abs(int(g1)-int(c2.G)) <= int(tolerance) &&
		abs(int(b1)-int(c2.B)) <= int(tolerance)
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// validateHexColor validates a hex color string format
func validateHexColor(hex string) error {
	// Remove # prefix if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	if len(hex) != 6 {
		return fmt.Errorf("must be 6 characters (e.g., #FFFFFF or FFFFFF), got %d characters", len(hex))
	}

	// Check if all characters are valid hex digits
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("contains invalid character '%c', must be hexadecimal (0-9, a-f, A-F)", c)
		}
	}

	return nil
}

// parseHexColor converts a hex color string to color.RGBA
func parseHexColor(hex string) (color.RGBA, error) {
	// Remove # prefix if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	if len(hex) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex color format: %s", hex)
	}

	var r, g, b uint8
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("failed to parse hex color: %w", err)
	}

	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
