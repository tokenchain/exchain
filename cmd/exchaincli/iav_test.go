package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	acctype "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmtypes "github.com/okex/exchain/x/evm/types"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"os"
	"strings"
	"testing"
)

const (
	KEY_TOKEN_PAIR   = "s/k:token_pair/"
	KEY_DISTRIBUTION = "s/k:distribution/"
	KEY_GOV          = "s/k:gov/"
	KEY_UPGRADE      = "s/k:upgrade/"
	KEY_FARM         = "s/k:farm/"
	KEY_MAIN         = "s/k:main/"
	KEY_TOKEN        = "s/k:token/"
	KEY_ORDER        = "s/k:order/"
	KEY_MINT         = "s/k:mint/"
	KEY_ACC          = "s/k:acc/"
	KEY_SUPPLY       = "s/k:supply/"
	KEY_DEX          = "s/k:dex/"
	KEY_AMMSWAP      = "s/k:ammswap/"
	KEY_EVIDENCE     = "s/k:evidence/"
	KEY_EVM          = "s/k:evm/"
	KEY_LOCK         = "s/k:lock/"
	KEY_PARAMS       = "s/k:params/"
	KEY_STAKING      = "s/k:staking/"
	KEY_SLASHING     = "s/k:slashing/"
)

func TestBBB(t *testing.T) {
	bz, err := hex.DecodeString("be5b26f40a013012013018012201302a423078303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303201303a01304201304a01305201305a01306201306a022d3172022d31")
	fmt.Println(err)
	var config evmtypes.ChainConfig
	cdc.MustUnmarshalBinaryBare(bz, &config)
	fmt.Printf("chaincofig:%s\n", config.String())
	fmt.Println(len("aAEc7660B2efe02ff23082a2C5349127D51Bba71"))
}

func TestAAA(t *testing.T) {
	iaviewer("/Users/shaoyunzhan/Documents/wp/go_wp/okexchainProjects/tools/_cache_evm/data/application.db", KEY_ACC, 10, 10000)
}

func iaviewer(dataDir string, module string, version int, cacheSize int) {
	fmt.Println(dataDir, module, version, cacheSize)
	tree, err := ReadTree(dataDir, version, []byte(module), cacheSize)
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
	KEY_EVM: evmPrintKey,
	KEY_ACC: accPrintKey,
}

func accPrintKey(key []byte, value []byte) {
	if key[0] == acctype.AddressStoreKeyPrefix[0] {
		var acc exported.Account
		bz := value
		cdc.MustUnmarshalBinaryBare(bz, &acc)
		fmt.Printf("adress:%s;account:%s\n", hex.EncodeToString(key[1:]), acc.String())
		return
	} else if bytes.Equal(key, acctype.GlobalAccountNumberKey) {
		fmt.Printf("%s:%s\n", string(key), hex.EncodeToString(value))
	} else {
		printKey := parseWeaveKey(key)
		digest := sha256.Sum256(value)
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
		digest := sha256.Sum256(value)
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
