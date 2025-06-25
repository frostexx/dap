package wallet

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

var ErrUnAuthorized = errors.New("unauthorized")

func getTxErrorFromResultXdr(resultXdr string) error {
	var txResult xdr.TransactionResult
	if err := xdr.SafeUnmarshalBase64(resultXdr, &txResult); err != nil {
		return fmt.Errorf("failed to decode result XDR: %w", err)
	}

	// Transaction-level error
	if txResult.Result.Code != xdr.TransactionResultCodeTxSuccess {
		return fmt.Errorf("transaction failed with code: %s", txResult.Result.Code.String())
	}

	if txResult.Result.Results == nil {
		return fmt.Errorf("transaction succeeded but no operation results returned")
	}

	for i, opResult := range *txResult.Result.Results {
		switch opResult.Tr.Type {
		case xdr.OperationTypePayment:
			if opResult.Tr.PaymentResult == nil {
				return fmt.Errorf("operation %d: missing payment result", i)
			}
			code := opResult.Tr.PaymentResult.Code
			if code != xdr.PaymentResultCodePaymentSuccess {
				return fmt.Errorf("operation %d failed: %s", i, code.String())
			}

		case xdr.OperationTypeClaimClaimableBalance:
			if opResult.Tr.ClaimClaimableBalanceResult == nil {
				return fmt.Errorf("operation %d: missing claim claimable balance result", i)
			}
			code := opResult.Tr.ClaimClaimableBalanceResult.Code
			if code != xdr.ClaimClaimableBalanceResultCodeClaimClaimableBalanceSuccess {
				return fmt.Errorf("operation %d failed: %s", i, code.String())
			}

		default:
			return fmt.Errorf("operation %d has unsupported type: %s", i, opResult.Tr.Type.String())
		}
	}

	return nil
}

func (w *Wallet) Transfer(kp *keypair.Full, amount float64, address string) error {
	//const address = "GDBGPLINZX67V4BLRS4EW3KP3BZ4QZ7CYUZHISD3SZ5JBM6K6BT63XMC"

	account, err := w.GetAccount(kp)
	if err != nil {
		return err
	}

	paymentOp := txnbuild.Payment{
		Destination: address,
		Amount:      strconv.FormatFloat(amount, 'f', -1, 64),
		Asset:       txnbuild.NativeAsset{},
	}

	txParams := txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{&paymentOp},
		BaseFee:              1_000_000,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	}

	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return fmt.Errorf("error building transaction: %v", err)
	}

	signedTx, err := tx.Sign(w.networkPassphrase, kp)
	if err != nil {
		return fmt.Errorf("error signing transaction: %v", err)
	}

	resp, err := w.client.SubmitTransaction(signedTx)
	if err != nil {
		return fmt.Errorf("error submitting transaction: %v", err)
	}

	if !resp.Successful {
		err = getTxErrorFromResultXdr(resp.ResultXdr)
		fmt.Println(err)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("whataaa")
	}

	//fmt.Println(resp.Successful, "---", resp.Hash, "---")

	return nil
}

func (w *Wallet) WithdrawClaimableBalance(kp *keypair.Full, amountStr, balanceID, address string) (string, float64, error) {
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return "", 0, fmt.Errorf("error formatting amount: %s", err.Error())
	}
	amount = amount - 0.01

	hash, err := w.ClaimAndWithdraw(kp, amount, balanceID, address)
	if err != nil {
		return "", 0, fmt.Errorf("error claiming and withdrawing: %v", err)
	}

	return hash, amount, nil
}

func (w *Wallet) CreateClaimable(kp *keypair.Full, recipientAddress string, amount float64) (string, error) {
	senderAccount, err := w.GetAccount(kp)
	if err != nil {
		return "", err
	}

	t := time.Now().Add(10 * time.Minute)
	claimant := txnbuild.Claimant{
		Destination: recipientAddress,
		Predicate:   txnbuild.NotPredicate(txnbuild.BeforeAbsoluteTimePredicate(t.Unix())),
	}

	createOp := txnbuild.CreateClaimableBalance{
		Asset:        txnbuild.NativeAsset{},
		Amount:       fmt.Sprintf("%.2f", amount),
		Destinations: []txnbuild.Claimant{claimant},
	}

	txParams := txnbuild.TransactionParams{
		SourceAccount:        &senderAccount,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{&createOp},
		BaseFee:              1_000_000, //txnbuild.MinBaseFee,
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

	return resp.Hash, nil
}
