package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ClientOptions struct{}

type Client struct {
	client *http.Client
	token  string
	host   string
}

func NewClient(host, token string) *Client {
	return &Client{
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		token: token,
		host:  host,
	}
}

func (c *Client) Get(url string, options ...ClientOptions) (*http.Response, error) {
	return c.client.Get(fmt.Sprintf("%s%s", c.host, url))
}

func (c *Client) Post(url string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return c.client.Post(fmt.Sprintf("%s%s", c.host, url),
		"application/json",
		bytes.NewBuffer(jsonBody))
}
