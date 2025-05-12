package browser

import (
	"context"
	"fmt"
	"log"
	"os"
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
	Bsleep(seconds int)
}

func (cdp *ChromeDP) InitChromeDP() (ChromeDP_CTX, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.Flag("remote-debugging-address", "0.0.0.0"),
		chromedp.Flag("window-size", "1280,800"),
		chromedp.Flag("user-data-dir", os.Getenv("CHROMIUM_USER_DATA_DIR")),
	)
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	if allocCtx == nil || ctx == nil {
		return ChromeDP_CTX{}, fmt.Errorf("failed to initialize ChromeDP context")
	}

	return ChromeDP_CTX{
		AllocCtx:    allocCtx,
		Ctx:         ctx,
		CancelAlloc: cancelAlloc,
		CancelCtx:   cancelCtx,
	}, nil
}

func (cdp *ChromeDP) Run(actions ...chromedp.Action) error {
	err := chromedp.Run(cdp.Ctx, actions...)
	if err != nil {
		log.Printf("Error running chromedp actions: %v", err)
	}
	return err
}

func (cdp *ChromeDP) CloseChromeDP() {
	cdp.CancelCtx()
	cdp.CancelAlloc()
}

func (cdp *ChromeDP) Navigate(url string) error {
	if err := chromedp.Run(cdp.Ctx, chromedp.Navigate(url)); err != nil {
		log.Printf("Error navigating to %s: %v", url, err)
	}
	return nil
}

func (cdp *ChromeDP) Evaluate(expression string) (string, error) {
	var result string
	if err := chromedp.Run(cdp.Ctx, chromedp.Evaluate(expression, &result)); err != nil {
		log.Printf("Error evaluating expression %s: %v", expression, err)
		return "", err
	}
	return result, nil
}

func (cdp *ChromeDP) Bsleep(seconds int) {
	if seconds > 0 {
		chromedp.Sleep(time.Second * time.Duration(seconds)).Do(cdp.Ctx)
	}
}
