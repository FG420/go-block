package wallet

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const walletFile = "./tmp/wallets.json"

type Wallets struct {
	Wallets map[string]*Wallet
}

func CreateWallets() (*Wallets, error) {
	var wallets Wallets

	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFile()

	return &wallets, err
}

func (ws *Wallets) GetAddress(addr string) *Wallet {
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

	ws.Wallets[addr] = wallet
	return addr
}

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return nil
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var temp struct {
		Wallets map[string]*Wallet `json:"Wallets"`
	}

	err = json.Unmarshal(fileContent, &temp)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = temp.Wallets

	return nil
}

func (ws Wallets) SaveFile() {
	jsonData, err := json.Marshal(ws)
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, jsonData, 0666)
	if err != nil {
		log.Panic(err)
	}
}
