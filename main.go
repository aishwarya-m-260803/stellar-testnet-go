package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/network"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

func main() {

	// 1. Generate a new Stellar keypair
	account := keypair.MustRandom()

	publicKey := account.Address()
	secretKey := account.Seed()

	fmt.Println("=== Stellar Testnet Wallet ===")
	fmt.Println("Public Key:", publicKey)
	fmt.Println("Secret Key:", secretKey)

	// 2. Fund the account using Friendbot
	friendbotURL := "https://friendbot-testnet.stellar.org/"

	requestURL := friendbotURL + "?addr=" + url.QueryEscape(publicKey)

	resp, err := http.Get(requestURL)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Friendbot request failed:")
		fmt.Println(string(body))
		return
	}

	fmt.Println("\n=== Account Funded ===")
	fmt.Println(string(body))
	fmt.Println("\nYour Stellar Testnet account is ready!")

	// 3. Fetch sender account details from Horizon
	client := horizonclient.DefaultTestNetClient
	accountRequest := horizonclient.AccountRequest{AccountID: publicKey}

	horizonAccount, err := client.AccountDetail(accountRequest)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== Horizon Account Details ===")
	fmt.Println("Account ID:", horizonAccount.AccountID)
	fmt.Println("Sequence:", horizonAccount.Sequence)
	fmt.Println("Balances:")
	for _, balance := range horizonAccount.Balances {
		if balance.Asset.Type == "native" {
			fmt.Printf("  - Type: native (XLM), Balance: %s\n", balance.Balance)
		} else {
			fmt.Printf("  - Type: %s, Code: %s, Issuer: %s, Balance: %s\n",
				balance.Asset.Type, balance.Asset.Code, balance.Asset.Issuer, balance.Balance)
		}
	}

	// 4. Generate a second keypair as the receiver
	receiver := keypair.MustRandom()

	receiverPublicKey := receiver.Address()
	receiverSecretKey := receiver.Seed()

	fmt.Println("\n=== Receiver Account ===")
	fmt.Println("Public Key:", receiverPublicKey)
	fmt.Println("Secret Key:", receiverSecretKey)

	// Fund the receiver account using Friendbot so it exists on the ledger
	receiverFundbotURL := friendbotURL + "?addr=" + url.QueryEscape(receiverPublicKey)
	receiverResp, err := http.Get(receiverFundbotURL)
	if err != nil {
		panic(err)
	}
	defer receiverResp.Body.Close()

	if receiverResp.StatusCode != http.StatusOK {
		receiverBody, _ := io.ReadAll(receiverResp.Body)
		fmt.Println("Friendbot request for receiver failed:")
		fmt.Println(string(receiverBody))
		return
	}
	fmt.Println("Receiver account funded via Friendbot")

	// 5. Send 1 XLM from sender to receiver
	paymentOp := txnbuild.Payment{
		Destination: receiverPublicKey,
		Amount:      "1",
		Asset:       txnbuild.NativeAsset{},
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &horizonAccount,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{&paymentOp},
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewTimeout(300),
			},
		},
	)
	if err != nil {
		panic(err)
	}

	tx, err = tx.Sign(network.TestNetworkPassphrase, account)
	if err != nil {
		panic(err)
	}

	txResp, err := client.SubmitTransaction(tx)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== Payment Sent ===")
	fmt.Println("Transaction Hash:", txResp.Hash)
	fmt.Println("Successful:", txResp.Successful)
	fmt.Println("1 XLM sent from sender to receiver!")

	// 6. Fetch transaction history for the sender account
	txRequest := horizonclient.TransactionRequest{
		ForAccount:    publicKey,
		IncludeFailed: true,
		Order:         horizonclient.OrderDesc,
	}

	txHistory, err := client.Transactions(txRequest)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== Sender Transaction History ===")
	for i, record := range txHistory.Embedded.Records {
		fmt.Printf("\nTransaction #%d\n", i+1)
		fmt.Println("  Hash:           ", record.Hash)
		fmt.Println("  Ledger:         ", record.Ledger)
		fmt.Println("  Created At:     ", record.LedgerCloseTime)
		fmt.Println("  Source Account: ", record.Account)
		fmt.Println("  Successful:     ", record.Successful)
	}
}