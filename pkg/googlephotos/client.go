package googlephotos

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// HTTPClient interface to allow mocking or swapping standard client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Add user-agent to look more like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	return c.Do(req)
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// 1. Random pause (jitter) to avoid burst patterns
	// Sleep between 500ms and 1500ms
	jitter := time.Duration(500+rand.Intn(1000)) * time.Millisecond
	time.Sleep(jitter)

	var resp *http.Response
	var err error

	// 2. Retry Logic
	maxRetries := 5
	backoff := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err = c.client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			// Rate limited
			sleepTime := backoff * time.Duration(i+1)
			
			// Check Retry-After header
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
					sleepTime = seconds
				}
			}
			
			fmt.Printf("Rate limited (429). Retrying in %v...\n", sleepTime)
			time.Sleep(sleepTime)
			continue
		}
		
		// Success or other error
		return resp, nil
	}
	
	return resp, nil // Return last response (likely 429 if loop finished)
}
