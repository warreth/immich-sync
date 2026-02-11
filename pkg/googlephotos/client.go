package googlephotos

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type Client struct {
	client *http.Client
}

func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		client: &http.Client{
			Jar:           jar,
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
	req.Header.Set("User-Agent", userAgent)
	return c.Do(req)
}

// Head performs a lightweight HEAD request without jitter (used for content-type probing)
func (c *Client) Head(url string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return c.client.Do(req)
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	jitter := time.Duration(500+rand.Intn(1000)) * time.Millisecond
	time.Sleep(jitter)

	var resp *http.Response
	var err error

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

// Post performs a POST request with retry logic and cookie/session support
func (c *Client) Post(targetURL string, contentType string, body string) (*http.Response, error) {
	jitter := time.Duration(500+rand.Intn(1000)) * time.Millisecond
	time.Sleep(jitter)

	maxRetries := 5
	backoff := 5 * time.Second

	var resp *http.Response
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("POST", targetURL, strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Content-Type", contentType)

		resp, err = c.client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			sleepTime := backoff * time.Duration(i+1)
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
					sleepTime = seconds
				}
			}
			fmt.Printf("Rate limited (429). Retrying in %v...\n", sleepTime)
			time.Sleep(sleepTime)
			continue
		}

		return resp, nil
	}

	return resp, nil
}
