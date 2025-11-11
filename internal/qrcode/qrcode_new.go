package qrcode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	qrc "github.com/skip2/go-qrcode"
)

// GenerateWithSkip generates QR code using skip2/go-qrcode with manual logo compositing
func GenerateWithSkip(opts Options) ([]byte, error) {
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

	// Map error correction level
	var ecLevel qrc.RecoveryLevel

	// Force Highest error correction when logo is enabled
	if opts.IncludeLogo {
		ecLevel = qrc.Highest // 30% recovery - required for logo overlay
	} else {
		switch opts.ErrorCorrection {
		case "low", "L":
			ecLevel = qrc.Low
		case "medium", "M":
			ecLevel = qrc.Medium
		case "high", "Q":
			ecLevel = qrc.High
		case "highest", "H":
			ecLevel = qrc.Highest
		default:
			ecLevel = qrc.High
		}
	}

	// Generate QR code
	q, err := qrc.New(opts.Data, ecLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	// Parse colors
	fgColor, _ := parseHexColor(opts.ForegroundColor)
	bgColor, _ := parseHexColor(opts.BackgroundColor)

	q.ForegroundColor = fgColor
	q.BackgroundColor = bgColor

	// Generate QR code image
	qrImg := q.Image(opts.Size)

	// If logo is requested, composite it
	if opts.IncludeLogo {
		qrImg, err = compositeLogoOnQR(qrImg, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to composite logo: %w", err)
		}
	}

	// Handle transparent background if requested
	if opts.TransparentBackground {
		qrImg = makeImageTransparent(qrImg, bgColor)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrImg); err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	return buf.Bytes(), nil
}

// compositeLogoOnQR overlays a logo with safe zone onto the QR code
func compositeLogoOnQR(qrImg image.Image, opts Options) (image.Image, error) {
	logoPath := "internal/assets/logo.png"

	// Check if logo exists
	if _, err := os.Stat(logoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("logo file not found at %s", logoPath)
	}

	// Load logo
	logoFile, err := os.Open(logoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open logo: %w", err)
	}
	defer logoFile.Close()

	logo, err := png.Decode(logoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo: %w", err)
	}

	// Calculate logo size - matching reference (20-25% of QR width is standard)
	qrBounds := qrImg.Bounds()
	qrWidth := qrBounds.Dx()

	// Logo should be about 18% of QR code width
	// With 30% padding, total safe zone = 18% * (1 + 0.30*2) = 18% * 1.6 = 28.8% of QR width
	// Area coverage = (28.8%)^2 ≈ 8.3% which is well under 30% error correction limit
	logoTargetSize := qrWidth * 18 / 100

	// Resize logo
	logo = resizeImage(logo, logoTargetSize, logoTargetSize)

	// Recolor logo if requested
	if opts.LogoColor != "" {
		targetColor, _ := parseHexColor(opts.LogoColor)
		logo = recolorImage(logo, targetColor)
	}

	// Create safe zone - square/rectangular white border around logo (30% padding)
	logoBounds := logo.Bounds()
	logoWidth := logoBounds.Dx()
	logoHeight := logoBounds.Dy()

	// Add 30% padding on each side for clear safe zone
	padding := (logoWidth * 3) / 10
	safeZoneWidth := logoWidth + (padding * 2)
	safeZoneHeight := logoHeight + (padding * 2)

	// Parse background color for safe zone
	bgColor, _ := parseHexColor(opts.BackgroundColor)

	// Create image with logo + square safe zone
	logoWithSafeZone := image.NewRGBA(image.Rect(0, 0, safeZoneWidth, safeZoneHeight))

	// Fill with background color (rectangular safe zone like reference)
	draw.Draw(logoWithSafeZone, logoWithSafeZone.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Draw logo in center
	logoOffset := image.Pt(padding, padding)
	draw.Draw(logoWithSafeZone, image.Rectangle{
		Min: logoOffset,
		Max: logoOffset.Add(logoBounds.Size()),
	}, logo, logoBounds.Min, draw.Over)

	// Composite onto QR code (centered)
	result := image.NewRGBA(qrBounds)
	draw.Draw(result, qrBounds, qrImg, qrBounds.Min, draw.Src)

	// Calculate center position
	logoCenterX := (qrWidth - safeZoneWidth) / 2
	logoCenterY := (qrWidth - safeZoneHeight) / 2

	logoPos := image.Pt(logoCenterX, logoCenterY)
	draw.Draw(result, image.Rectangle{
		Min: logoPos,
		Max: logoPos.Add(image.Pt(safeZoneWidth, safeZoneHeight)),
	}, logoWithSafeZone, image.Point{}, draw.Over)

	return result, nil
}

// resizeImage resizes an image to target dimensions using bilinear interpolation
func resizeImage(img image.Image, targetWidth, targetHeight int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Maintain aspect ratio
	if srcWidth > srcHeight {
		targetHeight = (srcHeight * targetWidth) / srcWidth
	} else {
		targetWidth = (srcWidth * targetHeight) / srcHeight
	}

	resized := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Use bilinear interpolation for smoother scaling
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			// Map destination pixel to source coordinates
			srcX := float64(x) * float64(srcWidth) / float64(targetWidth)
			srcY := float64(y) * float64(srcHeight) / float64(targetHeight)

			// Get the four surrounding pixels
			x0 := int(srcX)
			y0 := int(srcY)
			x1 := x0 + 1
			y1 := y0 + 1

			// Clamp to image bounds
			if x1 >= srcWidth {
				x1 = srcWidth - 1
			}
			if y1 >= srcHeight {
				y1 = srcHeight - 1
			}

			// Get fractional parts
			fx := srcX - float64(x0)
			fy := srcY - float64(y0)

			// Get colors of the four surrounding pixels
			c00 := img.At(x0+bounds.Min.X, y0+bounds.Min.Y)
			c10 := img.At(x1+bounds.Min.X, y0+bounds.Min.Y)
			c01 := img.At(x0+bounds.Min.X, y1+bounds.Min.Y)
			c11 := img.At(x1+bounds.Min.X, y1+bounds.Min.Y)

			// Bilinear interpolation
			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			// Interpolate
			r := lerp2D(float64(r00), float64(r10), float64(r01), float64(r11), fx, fy)
			g := lerp2D(float64(g00), float64(g10), float64(g01), float64(g11), fx, fy)
			b := lerp2D(float64(b00), float64(b10), float64(b01), float64(b11), fx, fy)
			a := lerp2D(float64(a00), float64(a10), float64(a01), float64(a11), fx, fy)

			resized.Set(x, y, color.RGBA{
				R: uint8(uint32(r) >> 8),
				G: uint8(uint32(g) >> 8),
				B: uint8(uint32(b) >> 8),
				A: uint8(uint32(a) >> 8),
			})
		}
	}

	return resized
}

// lerp performs linear interpolation
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// lerp2D performs 2D bilinear interpolation
func lerp2D(c00, c10, c01, c11, fx, fy float64) float64 {
	top := lerp(c00, c10, fx)
	bottom := lerp(c01, c11, fx)
	return lerp(top, bottom, fy)
}

// recolorImage recolors all non-transparent pixels to target color
func recolorImage(img image.Image, targetColor color.RGBA) image.Image {
	bounds := img.Bounds()
	recolored := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldColor := img.At(x, y)
			_, _, _, a := oldColor.RGBA()

			if a > 0 {
				recolored.Set(x, y, color.RGBA{
					R: targetColor.R,
					G: targetColor.G,
					B: targetColor.B,
					A: uint8(a >> 8),
				})
			} else {
				recolored.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			}
		}
	}

	return recolored
}

// makeImageTransparent makes background color pixels transparent
func makeImageTransparent(img image.Image, bgColor color.RGBA) image.Image {
	bounds := img.Bounds()
	transparent := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldColor := img.At(x, y)
			r, g, b, a := oldColor.RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			// Check if color matches background (with tolerance)
			if colorMatches(r8, g8, b8, bgColor, 10) {
				transparent.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			} else {
				transparent.Set(x, y, color.RGBA{R: r8, G: g8, B: b8, A: uint8(a >> 8)})
			}
		}
	}

	return transparent
}
