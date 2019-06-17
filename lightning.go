package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	macaroon "gopkg.in/macaroon.v2"

	"github.com/decred/dcrlnd/lnrpc"
	"github.com/decred/dcrlnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newDcrlndClient(lndNode, tlsCertPath, macaroonPath string) (lnrpc.LightningClient, error) {

	// First attempt to establish a connection to lnd's RPC sever.
	creds, err := credentials.NewClientTLSFromFile(tlsCertPath, "")
	if err != nil {
		return nil, fmt.Errorf("unable to read cert file: %v", err)
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	// Load the specified macaroon file.
	macBytes, err := ioutil.ReadFile(macaroonPath)
	if err != nil {
		return nil, err
	}
	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return nil, err
	}

	// Now we append the macaroon credentials to the dial options.
	opts = append(
		opts,
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	)

	conn, err := grpc.Dial(lndNode, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to dial to lnd's gRPC server: %v", err)
	}

	// If we're able to connect out to the lnd node, then we can start up
	// the faucet safely.
	lnd := lnrpc.NewLightningClient(conn)

	return lnd, nil
}

// subscribeToInvoiceEvents creates a handler that listens for events regarding
// invoices and displays them.
//
// This is using the global lnClient, which is an ugly design but sufficient
// for demo purposes.
func subscribeToInvoiceEvents() error {

	// Create the streaming request.
	stream, err := lnClient.SubscribeInvoices(context.Background(), &lnrpc.InvoiceSubscription{})
	if err != nil {
		return err
	}

	// Create a go-routine that reads events from the stream and processes
	// them, one by one.
	go func() {
		for {
			inv, err := stream.Recv()
			if err == io.EOF {
				// End of data.
				return
			}
			if err != nil {
				fmt.Println("Error in stream:", err)
			}

			// TODO: Since this is a goroutine, all accesses to
			// waitingInvoices need to be protected by a mutex or
			// similar.
			if _, isWaiting := waitingInvoices[inv.AddIndex]; !isWaiting {
				// Not an invoice from this server node.
				continue
			}

			if !inv.Settled {
				// Invoice hasn't been settled yet.
				continue
			}

			fmt.Printf("Received payment for invoice %d rhash %064x\n", inv.AddIndex, inv.RHash)
		}
	}()

	return nil

}
