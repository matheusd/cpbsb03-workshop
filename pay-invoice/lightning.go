package main

import (
	"fmt"
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
