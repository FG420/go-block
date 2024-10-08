package wallet

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/FG420/go-block/handlers"
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

// Json Encoding
func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return nil
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileContent, ws)
	if err != nil {
		return err
	}

	return nil
}

func (ws *Wallets) SaveFile() {
	jsonData, err := json.Marshal(ws)
	handlers.HandleErr(err)

	err = os.WriteFile(walletFile, jsonData, 0644)
	handlers.HandleErr(err)
}

// Gob Encoding
// func (ws *Wallets) LoadFile() error {
// 	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
// 		return nil
// 	}

// 	file, err := os.Open(walletFile)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	decoder := gob.NewDecoder(file)
// 	err = decoder.Decode(ws)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (ws *Wallets) SaveFile() error {
// 	file, err := os.OpenFile(walletFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	encoder := gob.NewEncoder(file)
// 	err = encoder.Encode(ws)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
