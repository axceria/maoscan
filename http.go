package main

import (
	"fmt"
	"h12.io/socks"
	"log"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Proxy struct {
	ips                  map[string]struct{}
	targetSites          []string
	httpStatusValidation bool
	timeout              time.Duration
	maxHttpThreads       int64

	openHttpThreads int64
	mu              sync.Mutex
}

var Proxies = &Proxy{
	// in work
	targetSites: []string{"https://google.com", "https://cloudflare.com"},

	httpStatusValidation: false,
	// now cfg file
	timeout:        time.Second * 5,
	maxHttpThreads: int64(config.HttpThreads),
	ips:            make(map[string]struct{}),
}

func (p *Proxy) WorkerThread() {
	for {
		for atomic.LoadInt64(&p.openHttpThreads) < int64(config.HttpThreads) {
			p.mu.Lock()
			for proxy, _ := range p.ips {
				if strings.ToLower(config.ProxyType) == "http" {
					go p.CheckProxyHTTP(proxy)
				} else if strings.ToLower(config.ProxyType) == "socks4" {
					go p.CheckProxySocks4(proxy)
				} else if strings.ToLower(config.ProxyType) == "socks5" {
					go p.CheckProxySocks5(proxy)
				} else if strings.ToLower(config.ProxyType) == "all" {
					go p.CheckAllProxyType(proxy)
				} else {
					log.Fatalln("Supported proxy types: http|socks4|socks5|all")
				}
				delete(p.ips, proxy)
				break
			}
			p.mu.Unlock()

		}
		time.Sleep(time.Millisecond * 100)
	}
}

func (p *Proxy) CheckProxyHTTP(proxy string) {
	atomic.AddInt64(&p.openHttpThreads, 1)
	defer func() {
		atomic.AddInt64(&p.openHttpThreads, -1)
		atomic.AddUint64(&checked, 1)
	}()

	var err error
	var proxyPort = *port
	s := strings.Split(proxy, ":")
	if len(s) > 1 {
		proxyPort, err = strconv.Atoi(strings.TrimSpace(s[1]))
		if err != nil {
			log.Println(err)
			return
		}
	}

	if len(s) > 1 {
		var err error
		proxyPort, err = strconv.Atoi(s[1])
		if err != nil {
			log.Println(err)
			return
		}
	}
	proxyUrl, err := url.Parse(fmt.Sprintf("http://%s:%d", s[0], proxyPort))
	if err != nil {
		log.Println(err)
		return
	}

	tr := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
		DialContext: (&net.Dialer{
			Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
			KeepAlive: time.Second,
			DualStack: true,
		}).DialContext,
	}

	client := http.Client{
		Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
		Transport: tr,
	}

	req, err := http.NewRequest("GET", config.CheckSite, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("user-agent", config.Headers.UserAgent)
	req.Header.Add("accept", config.Headers.Accept)
	req.Header.Add("accept-encoding","gzip, deflate, br")
	req.Header.Add("accept-language","en-US,en;q=0.9")

	res, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&proxyErr, 1)
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddUint64(&timeoutErr, 1)
			return
		}
		return
	}
	res.Body.Close()

	if res.StatusCode == 407 {
		// dropped due to authentication
		atomic.AddUint64(&statusCodeErr, 1)
	} else if res.StatusCode == 200 {		
		atomic.AddUint64(&success, 1)
		exporter.Add(fmt.Sprintf(1,"%s:%d", s[0], proxyPort))
		if config.PrintIps.Enabled {
			go PrintProxy(s[0], proxyPort)
		}
	} else {
		if res.Body != nil {
			defer res.Body.Close()

			// do not read all, or die
			limitReader := io.LimitReader(res.Body, 4096)
			body, err := ioutil.ReadAll(limitReader)
			if err != nil {
				atomic.AddUint64(&statusCodeErr, 1)
			} else {
				if strings.Contains(string(body), "html") {
					atomic.AddUint64(&success, 1)
					exporter.Add(fmt.Sprintf(1,"%s:%d", s[0], proxyPort))
					if config.PrintIps.Enabled {
						go PrintProxy(s[0], proxyPort)
					}
				} else {
					atomic.AddUint64(&statusCodeErr, 1)
				}
			}
		} else {
			atomic.AddUint64(&statusCodeErr, 1)
		}
	}
}

func (p *Proxy) CheckProxySocks4(proxy string) {
	atomic.AddInt64(&p.openHttpThreads, 1)
	defer func() {
		atomic.AddInt64(&p.openHttpThreads, -1)
		atomic.AddUint64(&checked, 1)
	}()

	var err error
	var proxyPort = *port
	s := strings.Split(proxy, ":")
	if len(s) > 1 {
		proxyPort, err = strconv.Atoi(strings.TrimSpace(s[1]))
		if err != nil {
			log.Println(err)
			return
		}
	}

	tr := &http.Transport{
		Dial: socks.Dial(fmt.Sprintf("socks4://%s:%d?timeout=%ds", s[0], proxyPort, config.Timeout.Socks4Timeout)),
	}

	client := http.Client{
		Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
		Transport: tr,
	}

	req, err := http.NewRequest("GET", config.CheckSite, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("user-agent", config.Headers.UserAgent)
	req.Header.Add("accept", config.Headers.Accept)
	req.Header.Add("accept-encoding","gzip, deflate, br")
	req.Header.Add("accept-language","en-US,en;q=0.9")

	res, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&proxyErr, 1)
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddUint64(&timeoutErr, 1)
			return
		}
		return
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		atomic.AddUint64(&statusCodeErr, 1)
	} else {
		if config.PrintIps.Enabled {
			go PrintProxy(s[0], proxyPort)
		}
		atomic.AddUint64(&success, 1)
		exporter.Add(fmt.Sprintf(2,"%s:%d", s[0], proxyPort))
	}
}

func (p *Proxy) CheckProxySocks5(proxy string) {
	atomic.AddInt64(&p.openHttpThreads, 1)
	defer func() {
		atomic.AddInt64(&p.openHttpThreads, -1)
		atomic.AddUint64(&checked, 1)
	}()

	var err error
	var proxyPort = *port
	s := strings.Split(proxy, ":")
	if len(s) > 1 {
		proxyPort, err = strconv.Atoi(strings.TrimSpace(s[1]))
		if err != nil {
			log.Println(err)
			return
		}
	}

	tr := &http.Transport{
		Dial: socks.Dial(fmt.Sprintf("socks5://%s:%d?timeout=%ds", s[0], proxyPort, config.Timeout.Socks4Timeout)),
	}

	client := http.Client{
		Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
		Transport: tr,
	}

	req, err := http.NewRequest("GET", config.CheckSite, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("user-agent", config.Headers.UserAgent)
	req.Header.Add("accept", config.Headers.Accept)
	req.Header.Add("accept-encoding","gzip, deflate, br")
	req.Header.Add("accept-language","en-US,en;q=0.9")

	res, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&proxyErr, 1)
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddUint64(&timeoutErr, 1)
			return
		}
		return
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		atomic.AddUint64(&statusCodeErr, 1)
	} else {
		if config.PrintIps.Enabled {
			go PrintProxy(s[0], proxyPort)
		}
		atomic.AddUint64(&success, 1)
		exporter.Add(fmt.Sprintf(3,"%s:%d", s[0], proxyPort))
	}
}

func (p *Proxy) CheckAllProxyType(proxy string) {
	numworkers := 3
	tasks := make(chan func(), numworkers)
	var wg sync.WaitGroup

	// start workers
	for i := 0; i < numworkers; i++ {
		go func() {
			for task := range tasks {
				task()
				wg.Done()
			}
		}()
	}

	wg.Add(numworkers)
	tasks <- func() {
		p.CheckProxyHTTP(proxy)
	}
	tasks <- func() {
		p.CheckProxySocks4(proxy)
	}
	tasks <- func() {
		p.CheckProxySocks5(proxy)
	}

	wg.wait()
	close(tasks)
}
