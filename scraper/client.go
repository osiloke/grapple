package scraper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
	// "github.com/sethgrid/pester"
	"golang.org/x/net/proxy"

	"github.com/PuerkitoBio/goquery"
)

type Client interface {
	Post(string, url.Values) (*http.Response, error)
	PostBytes(string, url.Values) ([]byte, error)
	Get(string) (*http.Response, error)
	GetBytes(string) ([]byte, error)
	SocksEnabled() bool
	GetDoc(string) (*goquery.Document, error)
	GetFind(string, string) (*goquery.Selection, error)
}
type DefaultClient struct {
	Socks5Proxy  string
	Encoding     string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	Retry        int
	socksEnabled bool
	Client       *http.Client
}

type HTTPError struct {
	code int
}

func (h HTTPError) Code() int {
	return h.code
}
func (h HTTPError) Error() string {
	return fmt.Sprintf("%d", h.code)
}

func NewDefaultClient(client *DefaultClient) (Client, error) {
	if client == nil {
		client = new(DefaultClient)
	}
	if client.DialTimeout == 0 {
		client.DialTimeout = time.Second * 10
	}
	if client.ReadTimeout == 0 {
		client.ReadTimeout = time.Second * 10
	}
	if client.Encoding == "" {
		client.Encoding = "utf-8"
	}
	if client.Retry == 0 {
		client.Retry = 3
	}
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, client.DialTimeout)
		},
		Proxy: http.ProxyFromEnvironment,
		ResponseHeaderTimeout: client.ReadTimeout,
	}
	if client.Socks5Proxy != "" {
		p, err := proxy.SOCKS5("tcp", client.Socks5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		client.socksEnabled = true
		transport.Dial = p.Dial
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client.Client = &http.Client{
		Transport: transport,
		Jar:       jar,
	}
	return client, nil
}
func (c *DefaultClient) Post(url string, form url.Values) (*http.Response, error) {
	retry := c.Retry
	for {
		if req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode())); err == nil {
			req.Header.Set("content-type", "application/x-www-form-urlencoded")
			req.Header.Add("User-Agent", `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.27 Safari/537.36`)
			resp, err := c.Client.Do(req)
			if err != nil {
				if retry == 0 {
					return nil, err
				}
				retry--
				continue
			}
			if resp.StatusCode == 200 {
				return resp, nil
			} else {
				return nil, HTTPError{resp.StatusCode}
			}
		} else {
			return nil, err
		}
	}
}

func (c *DefaultClient) PostBytes(url string, form url.Values) ([]byte, error) {
	retry := c.Retry
	for {
		resp, err := c.Client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if err != nil {
			if retry == 0 {
				return nil, err
			}
			retry--
			continue
		} else {
			defer resp.Body.Close()
			contents, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return contents, nil
		}
	}
}
func (c *DefaultClient) Get(url string) (*http.Response, error) {
	retry := c.Retry
	for {
		if req, err := http.NewRequest("GET", url, nil); err == nil {
			req.Header.Add("User-Agent", `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.27 Safari/537.36`)
			resp, err := c.Client.Do(req)
			if err != nil {
				if retry == 0 {
					return nil, err
				}
				retry--
				continue
			}
			if resp.StatusCode == 200 {
				return resp, nil
			} else {
				return nil, HTTPError{resp.StatusCode}
			}
		} else {
			return nil, err
		}
	}
}

func (c *DefaultClient) GetBytes(url string) ([]byte, error) {
	retry := c.Retry
	for {
		resp, err := c.Client.Get(url)
		if err != nil {
			if retry == 0 {
				return nil, err
			}
			retry--
			continue
		} else {
			defer resp.Body.Close()
			contents, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return contents, nil
		}
	}
}

func (c *DefaultClient) SocksEnabled() bool {
	return c.socksEnabled
}

// TODO: Handle http error codes properly for 400, 429, 500
func (c *DefaultClient) GetDoc(url string) (*goquery.Document, error) {
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return goquery.NewDocumentFromResponse(resp)
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New("unable to retrieve doc " + resp.Status + " " + string(b))
	}
}

func (c *DefaultClient) GetFind(url string, selector string) (*goquery.Selection, error) {
	doc, err := c.GetDoc(url)
	if err != nil {
		return nil, err
	}
	return doc.Find(selector), nil
}
