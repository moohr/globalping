package pinger

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

type SimpleRemotePinger struct {
	Endpoint        string
	Request         SimplePingRequest
	ClientTLSConfig *tls.Config
}

func (sp *SimpleRemotePinger) Ping(ctx context.Context) <-chan PingEvent {
	evChan := make(chan PingEvent)
	go func() {
		vals := sp.Request.ToURLValues()
		urlObj, err := url.Parse(sp.Endpoint)
		if err != nil {
			evChan <- PingEvent{Error: err}
			return
		}
		urlObj.RawQuery = vals.Encode()

		client := &http.Client{}
		if sp.ClientTLSConfig != nil {
			client.Transport = &http.Transport{
				TLSClientConfig: sp.ClientTLSConfig,
				Proxy:           http.ProxyFromEnvironment,
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", urlObj.String(), nil)
		if err != nil {
			evChan <- PingEvent{Error: err}
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			evChan <- PingEvent{Error: err}
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				evChan <- PingEvent{Error: err}
				return
			}

			line := scanner.Bytes()
			log.Printf("[DBG] Scan line: %s", string(line))

			pingEVObj := new(PingEvent)
			if err := json.Unmarshal(line, pingEVObj); err != nil {
				evChan <- PingEvent{Error: err}
				return
			}
			evChan <- *pingEVObj
		}

	}()

	return evChan
}
