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
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/emiago/diago"
	"github.com/emiago/diago/examples"
	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// Have receiver running:
// gophone answer -l "127.0.0.1:5090"
//
// Run app:
// go run . sip:uas@127.0.0.1:5090
//
// Run dialer:
// gophone dial sip:bob@127.0.0.1

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	examples.SetupLogger()

	flag.Parse()

	// arguments
	localAddr := flag.Arg(0)

	coreProxyAddr:= flag.Arg(1)

	coreProxyUri := sip.Uri{}
	if err := sip.ParseUri("sip:" + coreProxyAddr + ":5060", &coreProxyUri); err != nil {
		return err
	}

	err := start(ctx, localAddr, coreProxy)
	if err != nil {
		slog.Error("PBX finished with error", "error", err)
	}
}

func start(ctx context.Context, localAddr string, coreProxyUri string) error {
	// Setup our main transaction user
	ua, err := sipgo.NewUA()
	if err != nil {
		panic(err)
	}

	srv, err := sipgo.NewServer(ua)
	if err != nil {
		panic(err)
	}

	d := diago.NewDiago(ua,
		WithTransport(
			Transport{
				Transport: "udp",
				BindHost:  localAddr,
				BindPort:  5060,
			},
		WithServer(srv)
	))

	srv.OnOptions(handleOptions)

	return d.Serve(ctx, func(inDialog *diago.DialogServerSession) {
		BridgeCall(d, inDialog, coreProxyUri)
	})
}

func handleOptions(req *sip.Request, tx sip.ServerTransaction) {

	src := req.Source()

	log.Println("OPTIONS from:", src)

	if strings.HasPrefix(src, allowedIP+":") {

		res := sip.NewResponseFromRequest(req, 200, "OK", body)

		_ = tx.Respond(res)
		return
	}

	log.Println("Received OPTIONS from a known valid source:", src)

}

func BridgeCall(d *diago.Diago, inDialog *diago.DialogServerSession, coreProxyUri sip.Uri) error {
	inDialog.Trying()  // Progress -> 100 Trying
	inDialog.Ringing() // Ringing -> 180 Response

	inCtx := inDialog.Context()
	ctx, cancel := context.WithTimeout(inCtx, 5*time.Second)
	defer cancel()

	bridge := diago.NewBridge()
	// Now answer our in dialog
	if err := inDialog.Answer(); err != nil {
		return err
	}
	if err := bridge.AddDialogSession(inDialog); err != nil {
		return err
	}

	outDialog, err := d.InviteBridge(ctx, coreProxyUri, &bridge, diago.InviteOptions{})
	if err != nil {
		t.Log("Dialing failed", err)
		return err
	}
	defer outDialog.Close()
	outCtx := outDialog.Context()

	defer inDialog.Hangup(inCtx)
	defer outDialog.Hangup(outCtx)

	// You can even easily detect who hangups
	select {
	case <-inCtx.Done():
	case <-outCtx.Done():
	}
	return nil
}
