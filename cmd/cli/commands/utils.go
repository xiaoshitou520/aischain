package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func decodeTransaction(tx *jsonrpc.Transaction) *TxResult {
	feeResult := strconv.FormatFloat(float64(tx.Fee)/float64(types.Coin), 'f', 4, 64)
	result := &TxResult{
		Execer:     tx.Execer,
		Payload:    tx.Payload,
		RawPayload: tx.RawPayload,
		Signature:  tx.Signature,
		Fee:        feeResult,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
	}

	if tx.Amount != 0 {
		result.Amount = strconv.FormatFloat(float64(tx.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	if tx.From != "" {
		result.From = tx.From
	}

	payloadValue, ok := tx.Payload.(map[string]interface{})["Value"].(map[string]interface{})
	if !ok {
		return result
	}
	for _, e := range [4]string{"Transfer", "Withdraw", "Genesis", "Hlock"} {
		if _, ok := payloadValue[e]; ok {
			if amtValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"]; ok {
				amt := amtValue.(float64) / float64(types.Coin)
				amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
				result.Payload.(map[string]interface{})["Value"].(map[string]interface{})[e].(map[string]interface{})["amount"] = amtResult
				break
			}
		}
	}
	if _, ok := payloadValue["Miner"]; ok {
		if rwdValue, ok := result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"]; ok {
			rwd := rwdValue.(float64) / float64(types.Coin)
			rwdResult := strconv.FormatFloat(rwd, 'f', 4, 64)
			result.Payload.(map[string]interface{})["Value"].(map[string]interface{})["Miner"].(map[string]interface{})["reward"] = rwdResult
		}
	}

	return result
}

func decodeAccount(acc *types.Account, precision int64) *AccountResult {
	balanceResult := strconv.FormatFloat(float64(acc.GetBalance())/float64(precision), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(acc.GetFrozen())/float64(precision), 'f', 4, 64)
	accResult := &AccountResult{
		Addr:     acc.GetAddr(),
		Currency: acc.GetCurrency(),
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return accResult
}

func constructAccFromLog(l *jsonrpc.ReceiptLogResult, key string) *types.Account {
	var cur int32
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["currency"]; ok {
		cur = int32(tmp.(float32))
	}
	var bal int64
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["balance"]; ok {
		bal = int64(tmp.(float64))
	}
	var fro int64
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["frozen"]; ok {
		fro = int64(tmp.(float64))
	}
	var ad string
	if tmp, ok := l.Log.(map[string]interface{})[key].(map[string]interface{})["addr"]; ok {
		ad = tmp.(string)
	}
	return &types.Account{
		Currency: cur,
		Balance:  bal,
		Frozen:   fro,
		Addr:     ad,
	}
}

func decodeLog(rlog jsonrpc.ReceiptDataResult) *ReceiptData {
	rd := &ReceiptData{Ty: rlog.Ty, TyName: rlog.TyName}

	for _, l := range rlog.Logs {
		rl := &ReceiptLog{Ty: l.Ty, TyName: l.TyName, RawLog: l.RawLog}
		switch l.Ty {
		//case 1, 4, 111, 112, 113, 114:
		case types.TyLogErr, types.TyLogGenesis, types.TyLogNewTicket, types.TyLogCloseTicket, types.TyLogMinerTicket,
			types.TyLogTicketBind, types.TyLogPreCreateToken, types.TyLogFinishCreateToken, types.TyLogRevokeCreateToken,
			types.TyLogTradeSellLimit, types.TyLogTradeBuyMarket, types.TyLogTradeSellRevoke,
			types.TyLogTradeBuyLimit, types.TyLogTradeSellMarket, types.TyLogTradeBuyRevoke:
			rl.Log = l.Log
		//case 2, 3, 5, 11:
		case types.TyLogFee, types.TyLogTransfer, types.TyLogDeposit, types.TyLogGenesisTransfer,
			types.TyLogTokenTransfer, types.TyLogTokenDeposit:
			rl.Log = &ReceiptAccountTransfer{
				Prev:    decodeAccount(constructAccFromLog(l, "prev"), types.Coin),
				Current: decodeAccount(constructAccFromLog(l, "current"), types.Coin),
			}
		//case 6, 7, 8, 9, 10, 12:
		case types.TyLogExecTransfer, types.TyLogExecWithdraw, types.TyLogExecDeposit, types.TyLogExecFrozen, types.TyLogExecActive, types.TyLogGenesisDeposit:
			var execaddr string
			if tmp, ok := l.Log.(map[string]interface{})["execaddr"].(string); ok {
				execaddr = tmp
			}
			rl.Log = &ReceiptExecAccountTransfer{
				ExecAddr: execaddr,
				Prev:     decodeAccount(constructAccFromLog(l, "prev"), types.Coin),
				Current:  decodeAccount(constructAccFromLog(l, "current"), types.Coin),
			}
		case types.TyLogTokenExecTransfer, types.TyLogTokenExecWithdraw, types.TyLogTokenExecDeposit, types.TyLogTokenExecFrozen, types.TyLogTokenExecActive,
			types.TyLogTokenGenesisTransfer, types.TyLogTokenGenesisDeposit:
			var execaddr string
			if tmp, ok := l.Log.(map[string]interface{})["execaddr"].(string); ok {
				execaddr = tmp
			}
			rl.Log = &ReceiptExecAccountTransfer{
				ExecAddr: execaddr,
				Prev:     decodeAccount(constructAccFromLog(l, "prev"), types.TokenPrecision),
				Current:  decodeAccount(constructAccFromLog(l, "current"), types.TokenPrecision),
			}
			// EVM合约日志处理逻辑
		case types.TyLogCallContract:
			rl.Log = buildCallContractResult(l)
		case types.TyLogContractData:
			rl.Log = buildContractDataResult(l)
		case types.TyLogContractState:
			rl.Log = buildContractStateResult(l)
		default:
			fmt.Printf("---The log with vlaue:%d is not decoded --------------------\n", l.Ty)
			return nil
		}
		rd.Logs = append(rd.Logs, rl)
	}
	return rd
}

func buildCallContractResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.ReceiptEVMContract{}
	proto.Unmarshal(data, receipt)
	rlog := &types.ReceiptEVMContractCmd{Caller: receipt.Caller, ContractAddr: receipt.ContractAddr, ContractName: receipt.ContractName, UsedGas: receipt.UsedGas}
	rlog.Ret = common.ToHex(receipt.Ret)
	return rlog
}

func buildContractDataResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.EVMContractData{}
	proto.Unmarshal(data, receipt)
	rlog := &types.EVMContractDataCmd{Creator: receipt.Creator, Name: receipt.Name, Addr: receipt.Addr, Alias: receipt.Alias}
	rlog.Code = common.ToHex(receipt.Code)
	rlog.CodeHash = common.ToHex(receipt.CodeHash)
	return rlog
}

func buildContractStateResult(l *jsonrpc.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	receipt := &types.EVMContractState{}
	proto.Unmarshal(data, receipt)
	rlog := &types.EVMContractStateCmd{Nonce: receipt.Nonce, Suicided: receipt.Suicided}
	rlog.StorageHash = common.ToHex(receipt.StorageHash)
	if receipt.Storage != nil {
		rlog.Storage = make(map[string]string)
		for k, v := range receipt.Storage {
			rlog.Storage[k] = common.ToHex(v)
		}
	}
	return rlog
}

func SendToAddress(rpcAddr string, from string, to string, amount int64, note string, isToken bool, tokenSymbol string, isWithdraw bool) {
	amt := amount
	if isWithdraw {
		amt = -amount
	}
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amt, Note: note}
	if !isToken {
		params.IsToken = false
	} else {
		params.IsToken = true
		params.TokenSymbol = tokenSymbol
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcAddr, "Chain33.SendToAddress", params, &res)
	ctx.Run()
}

func CreateRawTx(to string, amount float64, note string, isWithdraw bool, isToken bool, tokenSymbol string, execName string) (string, error) {
	if amount < 0 {
		return "", types.ErrAmount
	}
	amountInt64 := int64(amount*1e4) * 1e4
	if execName != "" && !types.IsAllowExecName(execName) {
		return "", types.ErrExecNameNotMatch
	}
	var tx *types.Transaction
	if !isToken {
		transfer := &types.CoinsAction{}
		if !isWithdraw {
			if execName != "" {
				v := &types.CoinsAction_TransferToExec{TransferToExec: &types.CoinsTransferToExec{Amount: amountInt64, Note: note, ExecName: execName}}
				transfer.Value = v
				transfer.Ty = types.CoinsActionTransferToExec
			} else {
				v := &types.CoinsAction_Transfer{Transfer: &types.CoinsTransfer{Amount: amountInt64, Note: note}}
				transfer.Value = v
				transfer.Ty = types.CoinsActionTransfer
			}
		} else {
			v := &types.CoinsAction_Withdraw{Withdraw: &types.CoinsWithdraw{Amount: amountInt64, Note: note, ExecName: execName}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), To: to}
	} else {
		transfer := &types.TokenAction{}
		if !isWithdraw {
			v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{Withdraw: &types.CoinsWithdraw{Cointoken: tokenSymbol, Amount: amountInt64, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("token"), Payload: types.Encode(transfer), To: to}
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		return "", err
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
}

func GetExecAddr(exec string) (string, error) {
	if ok, err := isAllowExecName(exec); !ok {
		return "", err
	}

	addrResult := address.ExecAddress(exec)
	result := addrResult
	return result, nil
}

func isAllowExecName(exec string) (bool, error) {
	// exec name长度不能超过系统限制
	if len(exec) > address.MaxExecNameLength {
		return false, types.ErrExecNameNotAllow
	}
	// exec name中不允许有 "-"
	if strings.Contains(exec, "-") {
		return false, types.ErrExecNameNotAllow
	}
	if strings.HasPrefix(exec, "user.") {
		return true, nil
	}
	for _, e := range []string{"none", "coins", "hashlock", "retrieve", "ticket", "token", "trade"} {
		if exec == e {
			return true, nil
		}
	}
	return false, types.ErrExecNameNotAllow
}
