// Copyright (C) 2015 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package dialer

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/util"
	"golang.org/x/net/proxy"
)

var (
	proxyDialer proxy.Dialer
	usingProxy  bool
	noFallback  = os.Getenv("ALL_PROXY_NO_FALLBACK") != ""
)

type dialFunc func(network, addr string) (net.Conn, error)

func init() {
	l.SetDebug("dialer", strings.Contains(os.Getenv("STTRACE"), "dialer") || os.Getenv("STTRACE") == "all")

	proxy.RegisterDialerType("socks", socksDialerFunction)
	proxyDialer = getDialer(proxy.Direct)
	usingProxy = proxyDialer != proxy.Direct

	if usingProxy {
		http.DefaultTransport = &http.Transport{
			Dial:                Dial,
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		// Defer this, so that logging gets setup.
		go func() {
			time.Sleep(500 * time.Millisecond)
			l.Infoln("Proxy settings detected")
			if noFallback {
				l.Infoln("Proxy fallback disabled")
			}
		}()
	} else {
		go func() {
			time.Sleep(500 * time.Millisecond)
			l.Debugln("Dialer logging disabled, as no proxy was detected")
		}()
	}
}

func dialWithFallback(proxyDialFunc dialFunc, fallbackDialFunc dialFunc, network, addr string) (net.Conn, error) {
	conn, err := proxyDialFunc(network, addr)
	if err == nil {
		l.Debugf("Dialing %s address %s via proxy - success, %s -> %s", network, addr, conn.LocalAddr(), conn.RemoteAddr())
		SetTCPOptions(conn)
		return dialerConn{
			conn, newDialerAddr(network, addr),
		}, nil
	}
	l.Debugf("Dialing %s address %s via proxy - error %s", network, addr, err)

	if noFallback {
		return conn, err
	}

	conn, err = fallbackDialFunc(network, addr)
	if err == nil {
		l.Debugf("Dialing %s address %s via fallback - success, %s -> %s", network, addr, conn.LocalAddr(), conn.RemoteAddr())
		SetTCPOptions(conn)
	} else {
		l.Debugf("Dialing %s address %s via fallback - error %s", network, addr, err)
	}
	return conn, err
}

// This is a rip off of proxy.FromURL for "socks" URL scheme
func socksDialerFunction(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	var auth *proxy.Auth
	if u.User != nil {
		auth = new(proxy.Auth)
		auth.User = u.User.Username()
		if p, ok := u.User.Password(); ok {
			auth.Password = p
		}
	}

	return proxy.SOCKS5("tcp", u.Host, auth, forward)
}

// This is a rip off of proxy.FromEnvironment with a custom forward dialer
func getDialer(forward proxy.Dialer) proxy.Dialer {
	allProxy := os.Getenv("all_proxy")
	if len(allProxy) == 0 {
		return forward
	}

	proxyURL, err := url.Parse(allProxy)
	if err != nil {
		return forward
	}
	prxy, err := proxy.FromURL(proxyURL, forward)
	if err != nil {
		return forward
	}

	noProxy := os.Getenv("no_proxy")
	if len(noProxy) == 0 {
		return prxy
	}

	perHost := proxy.NewPerHost(prxy, forward)
	perHost.AddFromString(noProxy)
	return perHost
}

type contextDirectDialer struct {
	ctx context.Context
}

func (d *contextDirectDialer) Dial(network, addr string) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(d.ctx, network, addr)
}

type dialerConn struct {
	net.Conn
	addr net.Addr
}

func (c dialerConn) RemoteAddr() net.Addr {
	return c.addr
}

func newDialerAddr(network, addr string) net.Addr {
	netaddr, err := net.ResolveIPAddr(network, addr)
	if err == nil {
		return netaddr
	}
	return fallbackAddr{network, addr}
}

type fallbackAddr struct {
	network string
	addr    string
}

func (a fallbackAddr) Network() string {
	return a.network
}

func (a fallbackAddr) String() string {
	return a.addr
}

// Sort available addresses, preferring unspecified alddress.
func tcpAddrLess(i interface{}, j interface{}) bool {
	return util.AddressUnspecifiedLess(i.(*net.TCPAddr), j.(*net.TCPAddr))
}

func dialContextReusePort(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := net.Dialer{
		Control: ReusePortControl,
	}
	localAddrInterface := registry.Get(network, tcpAddrLess)
	if localAddrInterface != nil {
		dialer.LocalAddr = localAddrInterface.(*net.TCPAddr)
	}

	conn, err := dialer.DialContext(ctx, network, addr)
	if dialer.LocalAddr != nil {
		l.Debugf("dialContextReusePort dial %s %s via %s: %v", network, addr, dialer.LocalAddr, err)
	}
	return conn, err
}
