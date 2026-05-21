package main

import (
	"strings"

	"github.com/emiago/sipgo/sip"
)

// bridgeCopyHeaderNames are copied with sip.CopyHeaders when present on the inbound INVITE.
var bridgeCopyHeaderNames = []string{
	//"Route",
	"P-Asserted-Identity",
	"Diversion",
	"History-Info",
	"Refer-To",
	"Referred-By",
}

func bridgeCopyHeaders(in *sip.Request, out sip.Message) {
	for _, name := range bridgeCopyHeaderNames {
		sip.CopyHeaders(name, in, out)
	}

	seen := make(map[string]struct{})
	for _, h := range in.Headers() {
		name := h.Name()
		if !strings.HasPrefix(strings.ToLower(name), "X-") {
			continue
		}
		key := sip.HeaderToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		sip.CopyHeaders(name, in, out)
	}
}
