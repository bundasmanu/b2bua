// SPDX-License-Identifier: MPL-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024, Emir Aganovic

/* TODO: Two things should be added as dependencies
1- An argument defining the BindHost to be used (Transport) - must use the statefulset IP address from k8s passed via helm ENV var + entrypoint.sh argument definition on binary run
2- An argument defining the Core Server DNS expected to listen/send Requests - must use the Service DNS from k8s Core Server service, declared again as an ENV var passed via helm + entrypoint.sh argument definition on binary run
*/

package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/emiago/diago"
	"github.com/emiago/diago/examples"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

var (
	resolver        = newDNSResolver()
	localEndpoint   SIPEndpoint
	outboundProxy   SIPEndpoint
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	examples.SetupLogger()

	localAddrFlag := flag.String("local_addr", "udp:127.0.0.1:5060", "Local address to bind to. Format: [protocol:]host[:port]")
	outboundProxyFlag := flag.String("outbound_proxy_addr", "127.0.0.1:5080", "Outbound proxy address. Format: host[:port]")
	flag.Parse()

	localEndpoint = getSIPEndpoint(*localAddrFlag)
	outboundProxy = getSIPEndpoint("udp:" + *outboundProxyFlag)
	// Override port default for outbound proxy if not specified
	if *outboundProxyFlag != "" && !containsColon(*outboundProxyFlag) {
		outboundProxy.Port = 5080
	}

	err := start(ctx, localEndpoint)
	if err != nil {
		slog.Error("PBX finished with error", "error", err)
	}
}

func containsColon(s string) bool {
	for _, ch := range s {
		if ch == ':' {
			return true
		}
	}
	return false
}

func start(ctx context.Context, endpoint SIPEndpoint) error {
	ua, err := sipgo.NewUA()
	if err != nil {
		return err
	}

	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return err
	}

	d := diago.NewDiago(ua,
		diago.WithTransport(diago.Transport{
			Transport: endpoint.Protocol,
			BindHost:  endpoint.Host,
			BindPort:  endpoint.Port,
		}),
		diago.WithServer(srv),
	)

	srv.OnOptions(handleOptions)

	return d.Serve(ctx, func(inDialog *diago.DialogServerSession) {
		if err := BridgeCall(d, inDialog); err != nil {
			log.Println("bridge call failed:", err)
		}
	})
}

func handleOptions(req *sip.Request, tx sip.ServerTransaction) {
	src := req.Source()
	log.Println("OPTIONS from:", src)

	allowed, err := resolver.VerifySource(context.Background(), src, outboundProxy.Host)
	if err != nil {
		log.Println("failed to verify OPTIONS source", err)
		return
	}

	if !allowed {
		log.Println("OPTIONS source does not match resolved core proxy DNS", src)
		return
	}

	res := sip.NewResponseFromRequest(req, 200, "OK", "")
	_ = tx.Respond(res)
}

func BridgeCall(d *diago.Diago, inDialog *diago.DialogServerSession) error {
	inDialog.Trying()
	inDialog.Ringing()

	inCtx := inDialog.Context()
	ctx, cancel := context.WithTimeout(inCtx, 5*time.Second)
	defer cancel()

	bridge := diago.NewBridge()
	if err := inDialog.Answer(); err != nil {
		return err
	}
	if err := bridge.AddDialogSession(inDialog); err != nil {
		return err
	}

	targetUri, err := resolver.NextRoundRobinURI(ctx, outboundProxy.Host, outboundProxy.Port)
	if err != nil {
		return err
	}

	outDialog, err := d.InviteBridge(ctx, targetUri, &bridge, diago.InviteOptions{})
	if err != nil {
		return err
	}
	defer outDialog.Close()
	outCtx := outDialog.Context()

	defer inDialog.Hangup(inCtx)
	defer outDialog.Hangup(outCtx)

	select {
	case <-inCtx.Done():
	case <-outCtx.Done():
	}
	return nil
}

