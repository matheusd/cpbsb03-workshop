package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/decred/dcrlnd/lnrpc"
)

func indexHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-type", "text/html")
	writer.Write([]byte(`
	Decred LN Paywall Demo<br><br>

	<a href="/requestAccess">Request Access</a>
	`))
}

func requestAccessHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-type", "text/html")
	inv := &lnrpc.Invoice{
		Value: 1e7, // 0.1 dcr
	}

	// User requested access. Create an invoice for her to pay.
	resp, err := lnClient.AddInvoice(context.Background(), inv)
	if err != nil {
		fmt.Printf("Error generating invoice: %v", err)

		writer.Write([]byte(fmt.Sprintf(`
		Error generating invoice: %v
		`, err)))
		return
	}

	fmt.Printf("Adding invoice %d rhash %0x64x\n", resp.AddIndex, resp.RHash)
	waitingInvoices[resp.AddIndex] = struct{}{}

	// Return the invoice #, full payment request and add a link
	// for her to check if access has been granted to the paywalled area.
	// Given this is just a demo, we'll use the original rhash
	// as key to check if the invoice has been paid but on a real
	// production app this would probably be stored on a database or
	// cache and probably associated with a user account.
	rhash := fmt.Sprintf("%064x", resp.GetRHash())
	writer.Write([]byte(fmt.Sprintf(`
	Waiting for payment of invoice #%d<br><br>
	Pay the following to get access: <pre>
	%s
	</pre><br><br>

	<a href="/checkAccess?rhash=%s">Check access</a>
	`, resp.GetAddIndex(), resp.GetPaymentRequest(), rhash)))
}

func checkAccessHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-type", "text/html")

	// Lookup the invoice in the node, given the request rhash. We could
	// consult waitingInvoices map as an optimization prior to doing this.
	hash := &lnrpc.PaymentHash{
		RHashStr: request.FormValue("rhash"),
	}
	inv, err := lnClient.LookupInvoice(context.Background(), hash)
	if err != nil {
		// Some error (eg: invoice doesn't exist).
		writer.Write([]byte(fmt.Sprintf(`
		Error verifying invoice: %v
		`, err)))
		return
	}

	// Invoices expire after 1 hour (by default). In that case,
	// indicate the user needs to regenerate it.
	if inv.CreationDate+inv.Expiry < time.Now().Unix() {
		writer.Write([]byte(fmt.Sprintf("Invoice expired. Please generate a new one and try again.")))
		return
	}

	// State INVOICE_OPEN means it hasn't been paid yet.
	if inv.GetState() == lnrpc.Invoice_OPEN {
		fmt.Printf("Invoice %d still open\n", inv.AddIndex)
		writer.Write([]byte(fmt.Sprintf("Invoice %d still open. Please pay it to get access.<pre>\n%s</pre>",
			inv.GetAddIndex(), inv.GetPaymentRequest())))
		return
	}

	// If we've reached this point, it means the invoice has been paid.
	fmt.Printf("Invoice %d paid. User allowed to proceed!\n", inv.AddIndex)
	writer.Write([]byte(fmt.Sprintf(`
		Invoice %d has been paid. You may now proceed. Yay!!!
	`, inv.GetAddIndex())))
}
