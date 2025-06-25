package wallet

import (
	"fmt"
	"os"
	"pi/util"
	"strconv"

	hClient "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/txnbuild"
)

type Wallet struct {
	networkPassphrase string
	serverURL         string
	client            *hClient.Client
}

func New() *Wallet {
	client := hClient.DefaultPublicNetClient
	client.HorizonURL = os.Getenv("NET_URL")

	return &Wallet{
		networkPassphrase: os.Getenv("NET_PASSPHRASE"),
		serverURL:         os.Getenv("NET_URL"),
		client:            client,
	}
}

func (w *Wallet) GetAddress(kp *keypair.Full) string {
	return kp.Address()
}

func (w *Wallet) Login(seedPhrase string) (*keypair.Full, error) {
	kp, err := util.GetKeyFromSeed(seedPhrase)
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func (w *Wallet) GetAccount(kp *keypair.Full) (horizon.Account, error) {
	accReq := hClient.AccountRequest{AccountID: kp.Address()}
	account, err := w.client.AccountDetail(accReq)
	if err != nil {
		return horizon.Account{}, fmt.Errorf("error fetching account details: %v", err)
	}

	return account, nil
}

func (w *Wallet) GetAvailableBalance(kp *keypair.Full) (string, error) {
	account, err := w.GetAccount(kp)
	if err != nil {
		return "", err
	}

	return account.Balances[0].Balance, nil
}

func (w *Wallet) GetTransactions(kp *keypair.Full, limit uint) ([]operations.Operation, error) {
	opReq := hClient.OperationRequest{
		ForAccount: kp.Address(),
		Limit:      limit,
		Order:      hClient.OrderDesc,
	}
	ops, err := w.client.Operations(opReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching account operations: %v", err)
	}

	return ops.Embedded.Records, nil
}

func (w *Wallet) GetLockedBalances(kp *keypair.Full) ([]horizon.ClaimableBalance, error) {
	req := hClient.ClaimableBalanceRequest{
		Claimant: kp.Address(),
		Limit:    50,
	}

	res, err := w.client.ClaimableBalances(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching locked balances: %v", err)
	}

	return res.Embedded.Records, nil
}

func (w *Wallet) ClaimAndWithdraw(kp *keypair.Full, amount float64, balanceID, address string) (string, error) {
	account, err := w.GetAccount(kp)
	if err != nil {
		return "", err
	}

	claimOp := txnbuild.ClaimClaimableBalance{
		BalanceID: balanceID,
	}

	paymentOp := txnbuild.Payment{
		Destination: address,
		Amount:      strconv.FormatFloat(amount, 'f', -1, 64),
		Asset:       txnbuild.NativeAsset{},
	}

	txParams := txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{&claimOp, &paymentOp},
		BaseFee:              1_000_000,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	}

	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return "", fmt.Errorf("error building transaction: %v", err)
	}

	signedTx, err := tx.Sign(w.networkPassphrase, kp)
	if err != nil {
		return "", fmt.Errorf("error signing transaction: %v", err)
	}

	resp, err := w.client.SubmitTransaction(signedTx)
	if err != nil {
		return "", fmt.Errorf("error submitting transaction: %v", err)
	}

	if !resp.Successful {
		return "", fmt.Errorf("transaction failed")
	}

	return resp.Hash, nil
}
