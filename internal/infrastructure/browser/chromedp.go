package browser

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

// this packeges purpose is to wrap the chromedp package into an interface

type ChromeDP_CTX struct {
	AllocCtx    context.Context
	Ctx         context.Context
	CancelAlloc context.CancelFunc
	CancelCtx   context.CancelFunc
}
type ChromeDP struct {
	ChromeDP_CTX
	ChromeDP_Options []chromedp.ExecAllocatorOption
}
type ChromeDPInterface interface {
	// Initialize the ChromeDP context
	InitChromeDP() (ChromeDP_CTX, error)
	// Close the ChromeDP context
	CloseChromeDP()
	// Run the ChromeDP actions
	Run(actions ...chromedp.Action) error
	// Navigate to a URL
	Navigate(url string) error
	// Evaluate a JavaScript expression
	Evaluate(expression string) (string, error)
	// Sleep for a specified number of seconds
	Bsleep(seconds int) error
}

func (cdp *ChromeDP) InitChromeDP() (ChromeDP_CTX, error) {
	// Start with a clean state - close any existing contexts
	if cdp.Ctx != nil || cdp.AllocCtx != nil {
		cdp.CloseChromeDP()
	}

	// Create all new contexts
	baseCtx := context.Background()

	// Use Docker-friendly options for headless Chrome
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(baseCtx, opts...)
	if allocCtx == nil {
		log.Println("Failed to create allocator context")
		return ChromeDP_CTX{}, fmt.Errorf("failed to create allocator context")
	}

	// Create browser context with a timeout

	_, cancelTimeout := context.WithTimeout(allocCtx, 30*time.Second)
	defer cancelTimeout()

	// Create new browser context
	browserCtx, cancelCtx := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	if browserCtx == nil {
		cancelAlloc()
		log.Println("Failed to create browser context")
		return ChromeDP_CTX{}, fmt.Errorf("failed to create browser context")
	}

	// Test browser by navigating to about:blank
	log.Println("Starting browser...")
	err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank"))
	if err != nil {
		log.Printf("Failed to start browser: %v", err)
		cancelCtx()
		cancelAlloc()
		return ChromeDP_CTX{}, fmt.Errorf("failed to start browser: %w", err)
	}
	log.Println("Browser started successfully")

	// Store contexts
	cdp.AllocCtx = allocCtx
	cdp.Ctx = browserCtx
	cdp.CancelAlloc = cancelAlloc
	cdp.CancelCtx = cancelCtx

	return ChromeDP_CTX{
		AllocCtx:    allocCtx,
		Ctx:         browserCtx,
		CancelAlloc: cancelAlloc,
		CancelCtx:   cancelCtx,
	}, nil
}

func (cdp *ChromeDP) Run(actions ...chromedp.Action) error {
	// Check context validity
	if cdp.Ctx == nil {
		return fmt.Errorf("chromedp context is nil, call InitChromeDP first")
	}

	err := chromedp.Run(cdp.Ctx, actions...)
	if err != nil {
		log.Printf("Error running chromedp actions: %v", err)
	}
	return err
}

func (cdp *ChromeDP) CloseChromeDP() {
	// Add comprehensive safety checks
	if cdp == nil {
		return
	}

	// Cancel contexts safely
	if cdp.CancelCtx != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in CancelCtx: %v", r)
				}
			}()
			cdp.CancelCtx()
			cdp.CancelCtx = nil
		}()
	}

	if cdp.CancelAlloc != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in CancelAlloc: %v", r)
				}
			}()
			cdp.CancelAlloc()
			cdp.CancelAlloc = nil
		}()
	}

	// Clear all context references
	cdp.Ctx = nil
	cdp.AllocCtx = nil
	log.Println("ChromeDP resources have been safely released")
}

func (cdp *ChromeDP) Navigate(url string) error {
	if cdp == nil || cdp.Ctx == nil {
		log.Println("Cannot navigate: ChromeDP or context is nil")
		return fmt.Errorf("cannot navigate: ChromeDP not properly initialized")
	}

	log.Printf("Navigating to: %s", url)
	err := chromedp.Run(cdp.Ctx, chromedp.Navigate(url))
	if err != nil {
		log.Printf("Error navigating to %s: %v", url, err)
		return err
	}
	log.Printf("Successfully navigated to %s", url)
	return nil
}

func (cdp *ChromeDP) Evaluate(expression string) (string, error) {
	// Check if context is valid
	if cdp.Ctx == nil {
		return "", fmt.Errorf("chromedp context is nil, call InitChromeDP first")
	}

	var result string
	if err := chromedp.Run(cdp.Ctx, chromedp.Evaluate(expression, &result)); err != nil {
		log.Printf("Error evaluating expression %s: %v", expression, err)
		return "", err
	}
	return result, nil
}

func (cdp *ChromeDP) Bsleep(seconds int) error {
	// Check context validity
	if cdp.Ctx == nil {
		return fmt.Errorf("chromedp context is nil, call InitChromeDP first")
	}

	if seconds > 0 {
		return chromedp.Run(cdp.Ctx, chromedp.Sleep(time.Second*time.Duration(seconds)))
	}
	return nil
}
