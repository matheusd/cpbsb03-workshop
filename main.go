package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrlnd/lnrpc"
)

var (
	// lnClient is a reference to the main lightning client that connects
	// us to an LN node.
	lnClient lnrpc.LightningClient

	// waitingInvoices is a list of invoices waiting for payment. This is
	// used to track which invoices were added by this server instance and
	// to show paid invoices.
	waitingInvoices map[uint64]struct{} = make(map[uint64]struct{})
)

// orFatal is a helper function to reduce boilerplate when dealing with errors.
//
// NOTE: this is only really used in demos and prototypes. Real-world go
// programs usually handle errors within their respective calls.
func orFatal(err error) {
	if err == nil {
		return
	}

	fmt.Println(err)
	os.Exit(1)
}

func main() {
	// Connect to the dcrlnd node. This requires the node's address,
	// the tls cert and macaroon file.
	cli, err := newDcrlndClient(
		"localhost:10009",
		"/home/user/.dcrlnd/tls.cert",
		"/home/user/.dcrlnd/data/chain/decred/testnet/admin.macaroon")
	orFatal(err)

	// Get some basic info of the connected node.
	nodeInfo, err := cli.GetInfo(context.Background(),
		&lnrpc.GetInfoRequest{})
	orFatal(err)
	fmt.Printf("Connected to dcrln node %s\n", nodeInfo.GetIdentityPubkey())

	chanBal, err := cli.ChannelBalance(context.Background(), &lnrpc.ChannelBalanceRequest{})
	orFatal(err)
	fmt.Printf("Maximum inbound: %s across %d channels\n",
		dcrutil.Amount(chanBal.MaxInboundAmount),
		nodeInfo.NumActiveChannels)

	// Register the HTTP handlers.
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/requestAccess", requestAccessHandler)
	http.HandleFunc("/checkAccess", checkAccessHandler)

	// Store the dcrlnd client.
	lnClient = cli

	// Listen for invoice events.
	err = subscribeToInvoiceEvents()
	orFatal(err)

	// Start the HTPP server.
	bindAddr := "localhost:9090"
	fmt.Printf("Listening on %s\n", bindAddr)
	orFatal(http.ListenAndServe(bindAddr, nil))
}
