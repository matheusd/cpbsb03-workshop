package main

import (
	"context"
	"fmt"
	"os"

	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrlnd/lnrpc"
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
	if len(os.Args) < 2 {
		orFatal(fmt.Errorf("Please specify the payreq as argument"))
	}

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
	fmt.Printf("Maximum outbound: %s across %d channels\n",
		dcrutil.Amount(chanBal.MaxOutboundAmount),
		nodeInfo.NumActiveChannels)

	// Create the args to request the payment from the server.
	req := &lnrpc.SendRequest{
		PaymentRequest: os.Args[1],
	}

	// Pay the request and check the response.
	resp, err := cli.SendPaymentSync(context.Background(), req)
	orFatal(err)

	if resp.PaymentError != "" {
		orFatal(fmt.Errorf("Payment error: %s", resp.PaymentError))
	}

	fmt.Printf("Successuflly paid for invoice: %064x\n", resp.PaymentHash)
	fmt.Printf("Preimage was: %064x\n", resp.PaymentPreimage)
}
