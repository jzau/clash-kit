package clash

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Dreamacro/clash/adapter"
	"github.com/Dreamacro/clash/adapter/outboundgroup"
	"github.com/Dreamacro/clash/config"
	"github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/hub/executor"
	"github.com/Dreamacro/clash/log"
	"github.com/Dreamacro/clash/tunnel"
	"github.com/Dreamacro/clash/tunnel/statistic"
)

type Client interface {
	Traffic(up, down int64)
	Log(level, message string)
}

var (
	base   *config.Config
	client Client
)

func Setup(homeDir, config string, c Client) {
	client = c
	go fetchLogs()
	go fetchTraffic()
	constant.SetHomeDir(homeDir)
	constant.SetConfig("")
	cfg, err := executor.ParseWithBytes(([]byte)(config))
	if err != nil {
		panic(err)
	}
	base = cfg
	executor.ApplyConfig(base, true)
}

func GetConfigGeneral() []byte {
	if base == nil {
		return nil
	}
	data, _ := json.Marshal(base.General)
	return data
}

func PatchSelector(data []byte) bool {
	if base == nil {
		return false
	}
	mapping := make(map[string]string)
	err := json.Unmarshal(data, &mapping)
	if err != nil {
		return false
	}
	proxies := tunnel.Proxies()
	for name, proxy := range proxies {
		selected, exist := mapping[name]
		if !exist {
			continue
		}
		outbound, ok := proxy.(*adapter.Proxy)
		if !ok {
			continue
		}
		selector, ok := outbound.ProxyAdapter.(*outboundgroup.Selector)
		if !ok {
			continue
		}
		err := selector.Set(selected)
		if err == nil {
			return true
		}
	}
	return false
}

func fetchLogs() {
	ch := make(chan log.Event, 1024)
	sub := log.Subscribe()
	defer log.UnSubscribe(sub)
	go func() {
		for elm := range sub {
			l := elm.(log.Event)
			select {
			case ch <- l:
			default:
			}
		}
		close(ch)
	}()
	for l := range ch {
		if l.LogLevel < log.Level() {
			continue
		}
		client.Log(l.Type(), l.Payload)
	}
}

func fetchTraffic() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	t := statistic.DefaultManager
	for range tick.C {
		up, down := t.Now()
		client.Traffic(up, down)
	}
}

func GetProxyDelay(name, url string, timeout int16) int16 {
	proxies := tunnel.Proxies()
	proxy, exist := proxies[name]
	if !exist {
		return 0
	}

	delayChan := make(chan uint16, 1)
	errChan := make(chan error, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
		defer cancel()

		delay, err := proxy.URLTest(ctx, url)
		if err != nil {
			errChan <- err
		} else {
			delayChan <- delay
		}
	}()

	select {
	case delay := <-delayChan:
		return int16(delay)
	case <-errChan:
		return 0
	case <-time.After(time.Millisecond * time.Duration(timeout)):
		return 0
	}
}

func PatchTunnelMode(modeStr string) bool {
	if base == nil {
		return false
	}

	// Convert mode string to TunnelMode
	var mode tunnel.TunnelMode
	switch strings.ToLower(modeStr) {
	case "global":
		mode = tunnel.Global
	case "rule":
		mode = tunnel.Rule
	case "direct":
		mode = tunnel.Direct
	default:
		return false // invalid mode string
	}

	// Set the tunnel mode
	tunnel.SetMode(mode)
	return true
}
