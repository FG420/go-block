package wallet

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/FG420/go-block/utils"
)

const walletFile = "./tmp/wallets.data"

type WalletData struct {
	PublicKey []byte
}

type Wallets struct {
	Wallets map[string]*WalletData
}

func CreateWallets() (*Wallets, error) {
	var wallets Wallets

	wallets.Wallets = make(map[string]*WalletData)
	err := wallets.LoadFile()

	return &wallets, err
}

func (ws *Wallets) GetAddress(addr string) *WalletData {
	return ws.Wallets[addr]
}

func (ws *Wallets) GetAllAddresses() []string {
	var addrs []string

	for addr := range ws.Wallets {
		addrs = append(addrs, addr)
	}

	return addrs
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	addr := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[addr] = &WalletData{PublicKey: wallet.PublicKey}
	return addr
}

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return nil
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	var wallets Wallets
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets
	return nil
}

func (ws *Wallets) SaveFile() {
	var content bytes.Buffer

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	utils.HandleErr(err)

	err = os.WriteFile(walletFile, content.Bytes(), 0644)
	utils.HandleErr(err)
}
