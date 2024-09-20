package serialization

import (
	"github.com/bugnanetwork/bugnad/domain/bvm/state"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
)

func AccountToDBAccount(account *state.Account) *DbAccount {
	codeHash, _ := externalapi.NewDomainHashFromByteSlice(account.CodeHash)
	return &DbAccount{
		Nonce:           account.Nonce,
		CodeHash:        DomainHashToDbHash(codeHash),
		ScriptPublicKey: ScriptPublicKeyToDBScriptPublicKey(account.ScriptPublicKey),
	}
}

// DBKRC721ToKRC721 converts DbKRC721 to KRC721
func DBAccountToAccount(dbAccount *DbAccount) (*state.Account, error) {
	scriptPublicKey, err := DBScriptPublicKeyToScriptPublicKey(dbAccount.ScriptPublicKey)
	if err != nil {
		return nil, err
	}

	return &state.Account{
		Nonce:           dbAccount.Nonce,
		CodeHash:        dbAccount.CodeHash.GetHash(),
		ScriptPublicKey: scriptPublicKey,
	}, nil
}
