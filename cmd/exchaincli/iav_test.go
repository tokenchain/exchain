package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	acctypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	supplytypes "github.com/cosmos/cosmos-sdk/x/supply"
	evmtypes "github.com/okex/exchain/x/evm/types"
	slashingtypes "github.com/okex/exchain/x/slashing"
	tokentypes "github.com/okex/exchain/x/token/types"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"os"
	"strings"
	"testing"
)

const (
	KEY_DISTRIBUTION = "s/k:distribution/"
	KEY_GOV          = "s/k:gov/"
	KEY_MAIN         = "s/k:main/"
	KEY_TOKEN        = "s/k:token/"
	KEY_MINT         = "s/k:mint/"
	KEY_ACC          = "s/k:acc/"
	KEY_SUPPLY       = "s/k:supply/"
	KEY_EVM          = "s/k:evm/"
	KEY_PARAMS       = "s/k:params/"
	KEY_STAKING      = "s/k:staking/"
	KEY_SLASHING     = "s/k:slashing/"
)

func TestBBB(t *testing.T) {
	cdc.RegisterInterface((*govtypes.Content)(nil), nil)
	value, err := hex.DecodeString("86020A3F19360FC30A360A096F70656E206661726D12096F70656E206661726D1A1E0A046661726D12105969656C644E6174697665546F6B656E1A047472756510D8041001180322690A1E343834323637303737333339353436383137383239333636363837363537121E3335323139343233383036373835323233313134383633303331383239361A1E3335323139343233383036373835323233313134383633303331383239362201302A01303201302A0C08AC8C86800610B28BF9C502320C08AC8C86800610B28BF9C5023A1C0A036F6B741215313030303030303030303030303030303030303030420C08AC8C86800610B28BF9C5024A0C08E5A38680061086FB98B001")
	fmt.Println(err)
	var proposal govtypes.Proposal
	cdc.MustUnmarshalBinaryBare(value, &proposal)
	fmt.Printf("chaincofig:%s\n", proposal.String())
	addr, err := sdk.AccAddressFromBech32("0x382bb369d343125bfb2117af9c149795c6c65c50")
	fmt.Println(err)
	fmt.Println(addr)
	fmt.Println(len(addr.Bytes()))
}

func TestAAA(t *testing.T) {
	iaviewer("/Users/shaoyunzhan/Downloads/data/application.db", KEY_GOV, 5549923)
	//iaviewer("/Users/shaoyunzhan/Documents/wp/go_wp/okexchainProjects/tools/_cache_evm/data/application.db", KEY_TOKEN, 9)
}

func iaviewer(dataDir string, module string, version int) {
	tree, err := ReadTree(dataDir, version, []byte(module), 100000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading data: %s\n", err)
		os.Exit(1)
	}
	PrintKeys(module, tree)
	fmt.Printf("Hash: %X\n", tree.Hash())
	fmt.Printf("Size: %X\n", tree.Size())
}

func OpenDB(dir string) (dbm.DB, error) {
	switch {
	case strings.HasSuffix(dir, ".db"):
		dir = dir[:len(dir)-3]
	case strings.HasSuffix(dir, ".db/"):
		dir = dir[:len(dir)-4]
	default:
		return nil, fmt.Errorf("database directory must end with .db")
	}
	// TODO: doesn't work on windows!
	cut := strings.LastIndex(dir, "/")
	if cut == -1 {
		return nil, fmt.Errorf("cannot cut paths on %s", dir)
	}
	name := dir[cut+1:]
	db, err := dbm.NewGoLevelDB(name, dir[:cut])
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ReadTree loads an iavl tree from the directory
// If version is 0, load latest, otherwise, load named version
// The prefix represents which iavl tree you want to read. The iaviwer will always set a prefix.
func ReadTree(dir string, version int, prefix []byte, cacheSize int) (*iavl.MutableTree, error) {
	db, err := OpenDB(dir)
	if err != nil {
		return nil, err
	}
	if len(prefix) != 0 {
		db = dbm.NewPrefixDB(db, prefix)
	}

	tree, err := iavl.NewMutableTree(db, cacheSize)
	if err != nil {
		return nil, err
	}
	ver, err := tree.LoadVersion(int64(version))
	fmt.Printf("Got version: %d\n", ver)
	return tree, err
}

func PrintKeys(module string, tree *iavl.MutableTree) {
	fmt.Println("Printing all keys with hashed values (to detect diff)")
	tree.Iterate(func(key []byte, value []byte) bool {

		if impl, exit := printKeysDict[module]; exit {
			impl(key, value)
		} else {
			printKey := parseWeaveKey(key)
			digest := sha256.Sum256(value)
			fmt.Printf("  %s\n    %X\n", printKey, digest)
		}
		return false
	})
}

type (
	printKeys func(key []byte, value []byte)
)

var printKeysDict = map[string]printKeys{
	KEY_EVM:          evmPrintKey,
	KEY_ACC:          accPrintKey,
	KEY_PARAMS:       paramsPrintKey,
	KEY_STAKING:      stakingPrintKey,
	KEY_GOV:          govPrintKey,
	KEY_DISTRIBUTION: distributionPrintKey,
	//add
	KEY_SLASHING: slashingPrintKey,
	KEY_MAIN:     mainPrintKey,
	KEY_TOKEN:    tokenPrintKey,
	KEY_MINT:     mintPrintKey,
	KEY_SUPPLY:   supplyPrintKey,
}

func supplyPrintKey(key []byte, value []byte) {
	switch key[0] {
	case supplytypes.SupplyKey[0]:
		var supplyAmount sdk.Dec
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &supplyAmount)
		fmt.Printf("tokenSymbol:%s:info:%s\n", string(key[1:]), supplyAmount.String())
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

type MinterCustom struct {
	NextBlockToUpdate uint64       `json:"next_block_to_update" yaml:"next_block_to_update"` // record the block height for next year
	MintedPerBlock    sdk.DecCoins `json:"minted_per_block" yaml:"minted_per_block"`         // record the MintedPerBlock per block in this year
}

func mintPrintKey(key []byte, value []byte) {
	switch key[0] {
	case minttypes.MinterKey[0]:
		var minter MinterCustom
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &minter)
		fmt.Printf("minter:%v\n", minter)
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func tokenPrintKey(key []byte, value []byte) {
	switch key[0] {
	case tokentypes.TokenKey[0]:
		var token tokentypes.Token
		cdc.MustUnmarshalBinaryBare(value, &token)
		fmt.Printf("tokenName:%s:info:%s\n", string(key[1:]), token.String())
		return
	case tokentypes.TokenNumberKey[0]:
		var tokenNumber uint64
		cdc.MustUnmarshalBinaryBare(value, &tokenNumber)
		fmt.Printf("tokenNumber:%x\n", tokenNumber)
		return
	case tokentypes.PrefixUserTokenKey[0]:
		var token tokentypes.Token
		cdc.MustUnmarshalBinaryBare(value, &token)
		//address-token:tokenInfo
		fmt.Printf("%s-%s:token:%s\n", hex.EncodeToString(key[1:21]), string(key[21:]), token.String())
		return

	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func mainPrintKey(key []byte, value []byte) {
	if bytes.Equal(key, []byte("consensus_params")) {
		fmt.Printf("consensusParams:%s\n", hex.EncodeToString(value))
		return
	}
	printKey := parseWeaveKey(key)
	digest := hex.EncodeToString(value)
	fmt.Printf("  %s\n    %X\n", printKey, digest)
}

func slashingPrintKey(key []byte, value []byte) {
	switch key[0] {
	case slashingtypes.ValidatorSigningInfoKey[0]:
		var signingInfo slashingtypes.ValidatorSigningInfo
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &signingInfo)
		fmt.Printf("validatorAddr:%s:signingInfo:%s\n", hex.EncodeToString(key[1:]), signingInfo.String())
		return
	case slashingtypes.ValidatorMissedBlockBitArrayKey[0]:
		fmt.Printf("validatorMissedBlockAddr:%s:index:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case slashingtypes.AddrPubkeyRelationKey[0]:
		fmt.Printf("pubkeyAddr:%s:pubkey:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func distributionPrintKey(key []byte, value []byte) {
	switch key[0] {
	case distypes.FeePoolKey[0]:
		var feePool distypes.FeePool
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &feePool)
		fmt.Printf("feePool:%v\n", feePool)
		return
	case distypes.ProposerKey[0]:
		fmt.Printf("proposerKey:%s\n", hex.EncodeToString(value))
		return
	case distypes.DelegatorWithdrawAddrPrefix[0]:
		fmt.Printf("delegatorWithdrawAddr:%s:address:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case distypes.ValidatorAccumulatedCommissionPrefix[0]:
		var commission distypes.ValidatorAccumulatedCommission
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &commission)
		fmt.Printf("validatorAccumulatedAddr:%s:address:%s\n", hex.EncodeToString(key[1:]), commission.String())
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func govPrintKey(key []byte, value []byte) {
	switch key[0] {
	case govtypes.ProposalsKeyPrefix[0]:
		fmt.Printf("proposalId:%x;power:%x\n", binary.BigEndian.Uint64(key[1:]), hex.EncodeToString(value))
		return
	case govtypes.ActiveProposalQueuePrefix[0]:
		time, _ := sdk.ParseTimeBytes(key[1:])
		fmt.Printf("activeProposalEndTime:%x;proposalId:%x\n", time.String(), binary.BigEndian.Uint64(value))
		return
	case govtypes.InactiveProposalQueuePrefix[0]:
		time, _ := sdk.ParseTimeBytes(key[1:])
		fmt.Printf("inactiveProposalEndTime:%x;proposalId:%x\n", time.String(), binary.BigEndian.Uint64(value))
		return
	case govtypes.ProposalIDKey[0]:
		fmt.Printf("proposalId:%x\n", hex.EncodeToString(value))
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func stakingPrintKey(key []byte, value []byte) {
	switch key[0] {
	case stakingtypes.LastValidatorPowerKey[0]:
		var power int64
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &power)
		fmt.Printf("validatorAddress:%s;power:%x\n", hex.EncodeToString(key[1:]), power)
		return
	case stakingtypes.LastTotalPowerKey[0]:
		var power sdk.Int
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &power)
		fmt.Printf("lastTotolValidatorPower:%s\n", power.String())
		return
	case stakingtypes.ValidatorsKey[0]:
		var validator stakingtypes.Validator
		cdc.MustUnmarshalBinaryLengthPrefixed(value, &validator)
		fmt.Printf("validator:%s;info:%s\n", hex.EncodeToString(key[1:]), validator)
		return
	case stakingtypes.ValidatorsByConsAddrKey[0]:
		fmt.Printf("validatorConsAddrKey:%s;address:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case stakingtypes.ValidatorsByPowerIndexKey[0]:
		fmt.Printf("validatorPowerIndexKey:%s;address:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func paramsPrintKey(key []byte, value []byte) {
	fmt.Printf("%s:%s\n", string(key), string(value))
}

func accPrintKey(key []byte, value []byte) {
	if key[0] == acctypes.AddressStoreKeyPrefix[0] {
		var acc exported.Account
		bz := value
		cdc.MustUnmarshalBinaryBare(bz, &acc)
		fmt.Printf("adress:%s;account:%s\n", hex.EncodeToString(key[1:]), acc.String())
		return
	} else if bytes.Equal(key, acctypes.GlobalAccountNumberKey) {
		fmt.Printf("%s:%s\n", string(key), hex.EncodeToString(value))
	} else {
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

func evmPrintKey(key []byte, value []byte) {
	switch key[0] {
	case evmtypes.KeyPrefixBlockHash[0]:
		fmt.Printf("blockHash:%s;height:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case evmtypes.KeyPrefixBloom[0]:
		fmt.Printf("bloomHeight:%s;data:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case evmtypes.KeyPrefixCode[0]:
		fmt.Printf("code:%s;data:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case evmtypes.KeyPrefixStorage[0]:
		fmt.Printf("stroageHash:%s;keyHash:%s;data:%s\n", hex.EncodeToString(key[1:40]), hex.EncodeToString(key[41:]), hex.EncodeToString(value))
		return
	case evmtypes.KeyPrefixChainConfig[0]:
		bz := value
		var config evmtypes.ChainConfig
		cdc.MustUnmarshalBinaryBare(bz, &config)
		fmt.Printf("chainCofig:%s\n", config.String())
		return
	case evmtypes.KeyPrefixHeightHash[0]:
		fmt.Printf("height:%s;blockHash:%s\n", hex.EncodeToString(key[1:]), hex.EncodeToString(value))
		return
	case evmtypes.KeyPrefixContractDeploymentWhitelist[0]:
		fmt.Printf("whiteAddress:%s\n", hex.EncodeToString(key[1:]))
		return
	case evmtypes.KeyPrefixContractBlockedList[0]:
		fmt.Printf("blockedAddres:%s\n", hex.EncodeToString(key[1:]))
		return
	default:
		printKey := parseWeaveKey(key)
		digest := hex.EncodeToString(value)
		fmt.Printf("  %s\n    %X\n", printKey, digest)
	}
}

// parseWeaveKey assumes a separating : where all in front should be ascii,
// and all afterwards may be ascii or binary
func parseWeaveKey(key []byte) string {
	cut := bytes.IndexRune(key, ':')
	if cut == -1 {
		return encodeID(key)
	}
	prefix := key[:cut]
	id := key[cut+1:]
	fmt.Println("================", fmt.Sprintf("%s:%s", encodeID(prefix), encodeID(id)))
	return fmt.Sprintf("%s:%s", encodeID(prefix), encodeID(id))
}

// casts to a string if it is printable ascii, hex-encodes otherwise
func encodeID(id []byte) string {
	for _, b := range id {
		if b < 0x20 || b >= 0x80 {
			return strings.ToUpper(hex.EncodeToString(id))
		}
	}
	return string(id)
}
