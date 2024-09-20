package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/kaspanet/go-secp256k1"
	"github.com/pkg/errors"

	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/consensushashing"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/constants"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/subnetworks"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/transactionid"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
	utxopkg "github.com/bugnanetwork/bugnad/domain/consensus/utils/utxo"
	"github.com/bugnanetwork/bugnad/infrastructure/network/rpcclient"
	"github.com/bugnanetwork/bugnad/util"
)

func parsePrivateKeyInKeyPair(privateKeyHex string, prefix util.Bech32Prefix) (*secp256k1.SchnorrKeyPair, util.Address, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error parsing private key hex")
	}
	privateKey, err := secp256k1.DeserializeSchnorrPrivateKeyFromSlice(privateKeyBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error deserializing private key")
	}
	publicKey, err := privateKey.SchnorrPublicKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error generating public key")
	}

	pubKeySerialized, err := publicKey.Serialize()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error serializing public key")
	}

	pubKeyAddr, err := util.NewAddressPublicKey(pubKeySerialized[:], prefix)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error generating address from public key")
	}

	return privateKey, pubKeyAddr, nil
}

func printErrorAndExit(message string) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", message))
	os.Exit(1)
}

// Collect spendable UTXOs from address
func fetchAvailableUTXOs(client *rpcclient.RPCClient, address string) (map[appmessage.RPCOutpoint]*appmessage.RPCUTXOEntry, error) {
	getUTXOsByAddressesResponse, err := client.GetUTXOsByAddresses([]string{address})
	if err != nil {
		return nil, err
	}
	dagInfo, err := client.GetBlockDAGInfo()
	if err != nil {
		return nil, err
	}

	spendableUTXOs := make(map[appmessage.RPCOutpoint]*appmessage.RPCUTXOEntry, 0)
	for _, entry := range getUTXOsByAddressesResponse.Entries {
		if !isUTXOSpendable(entry, dagInfo.VirtualDAAScore) {
			continue
		}
		spendableUTXOs[*entry.Outpoint] = entry.UTXOEntry
	}
	return spendableUTXOs, nil
}

// Verify UTXO is spendable (check if a minimum of 10 confirmations have been processed since UTXO creation)
func isUTXOSpendable(entry *appmessage.UTXOsByAddressesEntry, virtualSelectedParentBlueScore uint64) bool {
	blockDAAScore := entry.UTXOEntry.BlockDAAScore
	if !entry.UTXOEntry.IsCoinbase {
		const minConfirmations = 10
		return blockDAAScore+minConfirmations < virtualSelectedParentBlueScore
	}
	coinbaseMaturity := uint64(100)
	return blockDAAScore+coinbaseMaturity < virtualSelectedParentBlueScore
}

func selectUTXOs(utxos map[appmessage.RPCOutpoint]*appmessage.RPCUTXOEntry, amountToSend uint64) (
	selectedUTXOs []*appmessage.UTXOsByAddressesEntry, selectedValue uint64, err error) {

	selectedUTXOs = []*appmessage.UTXOsByAddressesEntry{}
	selectedValue = uint64(0)

	for outpoint, utxo := range utxos {
		outpointCopy := outpoint
		selectedUTXOs = append(selectedUTXOs, &appmessage.UTXOsByAddressesEntry{
			Outpoint:  &outpointCopy,
			UTXOEntry: utxo,
		})
		selectedValue += utxo.Amount

		if selectedValue >= amountToSend {
			break
		}

		const maxInputs = 100
		if len(selectedUTXOs) == maxInputs {
			log.Infof("Selected %d UTXOs so sending the transaction with %d sompis instead "+
				"of %d", maxInputs, selectedValue, amountToSend)
			break
		}
	}

	return selectedUTXOs, selectedValue, nil
}

// Generate transaction data for initiating contract
func initiateContractTransaction(keyPair *secp256k1.SchnorrKeyPair, toAddress util.Address, selectedUTXOs []*appmessage.UTXOsByAddressesEntry,
	sompisToSend uint64, change uint64, fromAddress util.Address) (*appmessage.RPCTransaction, error) {

	// Generate transaction input from selectedUTXOs, collected from address query to Kaspad
	inputs := make([]*externalapi.DomainTransactionInput, len(selectedUTXOs))
	for i, utxo := range selectedUTXOs {
		outpointTransactionIDBytes, err := hex.DecodeString(utxo.Outpoint.TransactionID)
		if err != nil {
			return nil, err
		}
		outpointTransactionID, err := transactionid.FromBytes(outpointTransactionIDBytes)
		if err != nil {
			return nil, err
		}
		outpoint := externalapi.DomainOutpoint{
			TransactionID: *outpointTransactionID,
			Index:         utxo.Outpoint.Index,
		}
		utxoScriptPublicKeyScript, err := hex.DecodeString(utxo.UTXOEntry.ScriptPublicKey.Script)
		if err != nil {
			return nil, err
		}

		inputs[i] = &externalapi.DomainTransactionInput{
			PreviousOutpoint: outpoint,
			SigOpCount:       1,
			UTXOEntry: utxopkg.NewUTXOEntry(
				utxo.UTXOEntry.Amount,
				&externalapi.ScriptPublicKey{
					Script:  utxoScriptPublicKeyScript,
					Version: utxo.UTXOEntry.ScriptPublicKey.Version,
				},
				utxo.UTXOEntry.IsCoinbase,
				utxo.UTXOEntry.BlockDAAScore,
			),
		}
	}

	scriptPubkey, _ := txscript.PayToAddrScript(toAddress)

	// Generate transaction output to pay recipient address
	mainOutput := &externalapi.DomainTransactionOutput{
		Value:           sompisToSend,
		ScriptPublicKey: scriptPubkey,
	}

	// Generate ScriptPublicKey for change address
	fromScript, err := txscript.PayToAddrScript(fromAddress)
	if err != nil {
		return nil, err
	}

	// Generate array of Outputs and add "change address output", in case change have to be sent back to recipient address
	outputs := []*externalapi.DomainTransactionOutput{mainOutput}
	if change > 0 {
		changeOutput := &externalapi.DomainTransactionOutput{
			Value:           change,
			ScriptPublicKey: fromScript,
		}
		outputs = append(outputs, changeOutput)
	}

	// Generate transaction data (not yet signed)
	domainTransaction := &externalapi.DomainTransaction{
		Version:      constants.MaxTransactionVersion,
		Inputs:       inputs,
		Outputs:      outputs,
		LockTime:     0,
		SubnetworkID: subnetworks.SubnetworkIDNative,
		Gas:          0,
		Payload:      nil,
	}

	// Sign all inputs in transaction
	for i, input := range domainTransaction.Inputs {
		signatureScript, err := txscript.SignatureScript(domainTransaction, i, consensushashing.SigHashAll, keyPair,
			&consensushashing.SighashReusedValues{})
		if err != nil {
			return nil, err
		}
		input.SignatureScript = signatureScript
	}

	// Convert transaction into a RPC transaction, ready to be broadcasted
	rpcTransaction := appmessage.DomainTransactionToRPCTransaction(domainTransaction)
	return rpcTransaction, nil
}

// Generate transaction data for redeeming contract
func redeemContractTransaction(selectedUTXOToRedeem []*appmessage.UTXOsByAddressesEntry, redeemKeyPair *secp256k1.SchnorrKeyPair, recipientAddress util.Address, smartContractInput SmartContractInput) (*appmessage.RPCTransaction, error) {

	inputs := make([]*externalapi.DomainTransactionInput, len(selectedUTXOToRedeem))
	for i, utxo := range selectedUTXOToRedeem {
		outpointTransactionIDBytes, err := hex.DecodeString(utxo.Outpoint.TransactionID)
		if err != nil {
			return nil, err
		}
		outpointTransactionID, err := transactionid.FromBytes(outpointTransactionIDBytes)
		if err != nil {
			return nil, err
		}
		outpoint := externalapi.DomainOutpoint{
			TransactionID: *outpointTransactionID,
			Index:         utxo.Outpoint.Index,
		}
		utxoScriptPublicKeyScript, err := hex.DecodeString(utxo.UTXOEntry.ScriptPublicKey.Script)
		if err != nil {
			return nil, err
		}

		inputs[i] = &externalapi.DomainTransactionInput{
			PreviousOutpoint: outpoint,
			SigOpCount:       1,
			UTXOEntry: utxopkg.NewUTXOEntry(
				utxo.UTXOEntry.Amount,
				&externalapi.ScriptPublicKey{
					Script:  utxoScriptPublicKeyScript,
					Version: utxo.UTXOEntry.ScriptPublicKey.Version,
				},
				utxo.UTXOEntry.IsCoinbase,
				utxo.UTXOEntry.BlockDAAScore,
			),
		}
	}

	valueToRedeem := selectedUTXOToRedeem[0].UTXOEntry.Amount

	var feePerInput = uint64(30000)
	scriptPubkey, _ := txscript.PayToAddrScript(recipientAddress)
	outputs := []*externalapi.DomainTransactionOutput{{
		Value:           (valueToRedeem - uint64(feePerInput)*uint64(len(inputs))),
		ScriptPublicKey: scriptPubkey,
	}}

	// Generate transaction data (not yet signed)
	domainTransaction := &externalapi.DomainTransaction{
		Version:      constants.MaxTransactionVersion,
		Inputs:       inputs,
		Outputs:      outputs,
		LockTime:     0,
		SubnetworkID: subnetworks.SubnetworkIDNative,
		Gas:          0,
		Payload:      nil,
	}

	// Sign all inputs in transaction
	for i, input := range domainTransaction.Inputs {
		signatureScript, err := txscript.RawTxInSignature(domainTransaction, i, consensushashing.SigHashAll, redeemKeyPair, &consensushashing.SighashReusedValues{})
		if err != nil {
			return nil, err
		}

		redeemScript, err := smartContractInput.RedeemScript(signatureScript)
		if err != nil {
			return nil, err
		}

		input.SignatureScript = redeemScript
	}

	// Convert transaction into a RPC transaction, ready to be broadcasted
	rpcTransaction := appmessage.DomainTransactionToRPCTransaction(domainTransaction)
	return rpcTransaction, nil
}

// Broadcast transaction on the network
func sendTransaction(client *rpcclient.RPCClient, rpcTransaction *appmessage.RPCTransaction) (string, error) {
	submitTransactionResponse, err := client.SubmitTransaction(rpcTransaction, false)
	if err != nil {
		return "", errors.Wrapf(err, "error submitting transaction")
	}
	return submitTransactionResponse.TransactionID, nil
}

func initiateAddressScript(
	cfg *configFlags,
	client *rpcclient.RPCClient,
	myKeyPair *secp256k1.SchnorrKeyPair,
	myAddress util.Address,
	inputSmartContract SmartContractInput,
) (string, error) {
	//Fetch UTXOs from address
	availableUtxos, err := fetchAvailableUTXOs(client, myAddress.String())
	if err != nil {
		return "", errors.Wrap(err, "Available UTXOs can't be fetched")
	}

	//Define amount to send
	const balanceEpsilon = 10_000         // 10,000 sompi = 0.0001 kaspa
	const feeAmount = balanceEpsilon * 10 // use high fee amount, because can have a large number of inputs
	const sendAmount = balanceEpsilon * 1000
	totalSendAmount := uint64(sendAmount + feeAmount)

	//Select UTXOs matching Total Send amount
	selectedUTXOs, selectedValue, err := selectUTXOs(availableUtxos, totalSendAmount)
	if err != nil {
		return "", errors.Wrap(err, "UTXOs can't be selected")
	}
	if len(selectedUTXOs) == 0 {
		return "", fmt.Errorf("No UTXOs has been selected")
	}

	//Define change amount from selected UTXOs
	change := selectedValue - sendAmount - feeAmount

	// Create transaction
	rpcTransaction, err := initiateContractTransaction(myKeyPair, inputSmartContract.Address(), selectedUTXOs, sendAmount, change, myAddress)
	if err != nil {
		return "", errors.Wrap(err, "RpcTransaction can't be created")
	}

	//Broadcast transaction
	transactionID, err := sendTransaction(client, rpcTransaction)
	if err != nil {
		return "", errors.Wrap(err, "Transaction can't be correctly broadcasted")
	} else {
		log.Infof("Transaction has been successfully broadcasted: %s", transactionID)
	}

	return transactionID, nil
}

func redeemContract(client *rpcclient.RPCClient, transactionID string, redeemKeyPair *secp256k1.SchnorrKeyPair, redeemAddress util.Address, input SmartContractInput) (string, error) {

	log.Info("Starting deploy contract code operation...")

	var selectedUTXOToRedeem []*appmessage.UTXOsByAddressesEntry

	for {
		availableUtxos, err := fetchAvailableUTXOs(client, input.Address().String())
		if err != nil {
			return "", fmt.Errorf("Available UTXOs can't be fetched: %s", err)
		}

		selectedUTXOToRedeem, err = selectUTXOToRedeem(availableUtxos, transactionID)
		if err != nil {
			return "", fmt.Errorf("UTXOs can't be selected: %s", err)
		}
		if len(selectedUTXOToRedeem) == 0 {
			log.Error("No UTXOs has been selected, wait...")
			<-time.NewTicker(time.Second).C
			continue
		}

		break
	}

	//Select UTXOs matching contract TX
	// Create transaction
	rpcTransaction, err := redeemContractTransaction(selectedUTXOToRedeem, redeemKeyPair, redeemAddress, input)
	if err != nil {
		return "", fmt.Errorf("RpcTransaction can't be created: %s", err)
	}

	//Broadcast transaction
	redeemTransactionID, err := sendTransaction(client, rpcTransaction)
	if err != nil {
		return "", fmt.Errorf("Transaction can't be correctly broadcasted: %s", err)
	}

	return redeemTransactionID, nil
}

func selectUTXOToRedeem(availableUtxos map[appmessage.RPCOutpoint]*appmessage.RPCUTXOEntry, contractTxID string) (selectedUTXOs []*appmessage.UTXOsByAddressesEntry, err error) {

	selectedUTXOs = []*appmessage.UTXOsByAddressesEntry{}

	for outpoint, utxo := range availableUtxos {
		if outpoint.TransactionID == contractTxID {
			outpointCopy := outpoint
			selectedUTXOs = append(selectedUTXOs, &appmessage.UTXOsByAddressesEntry{
				Outpoint:  &outpointCopy,
				UTXOEntry: utxo,
			})
		}
	}

	return selectedUTXOs, nil
}
