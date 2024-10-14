package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/FG420/go-block/blockchain"
	"github.com/FG420/go-block/handlers"
	"github.com/FG420/go-block/wallet"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println(" getbalance -address ADDRESS - get the balance of the address")
	fmt.Println(" createbc -address ADDRESS - Creates a blockchain")
	fmt.Println(" printchain - Prints the blocks in the chain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT - Send amount from a user to another")
	fmt.Println(" createwallet - Creates a new Wallet")
	fmt.Println(" listaddrs - List the addresses in our wallet file ")
	fmt.Println(" reindexutxo - Rebuild the UTXO set ")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		b := iter.Next()

		fmt.Printf("Previous Hash: %x\n", b.PrevHash)
		fmt.Printf("Hash: %x\n", b.Hash)
		pow := blockchain.NewProof(b)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))

		for _, tx := range b.Transactions {
			fmt.Println(tx)
		}

		if len(b.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(addr string) {
	if !wallet.ValidateAddress(addr) {
		log.Panic("Address in not valid")
	}

	chain := blockchain.InitBlockChain(addr)
	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	chain.Database.Close()
	utxoSet.Reindex()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(addr string) {
	if !wallet.ValidateAddress(addr) {
		log.Panic("Address in not valid")
	}

	chain := blockchain.ContinueBlockChain(addr)
	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(addr))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTxOs := utxoSet.FindUTXO(pubKeyHash)

	for _, out := range UTxOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", addr, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		log.Panic("Address in not valid")
	}

	chain := blockchain.ContinueBlockChain(from)
	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	defer chain.Database.Close()

	log.Print("initialize new Transaction")
	tx := blockchain.NewTransaction(from, to, amount, &utxoSet)
	block := chain.AddBlock([]*blockchain.Transaction{tx})

	utxoSet.Update(block)
	fmt.Println("Success!")
}

func (cli *CommandLine) createWallet() {
	ws, _ := wallet.CreateWallets()
	addr := ws.AddWallet()

	ws.SaveFile()

	fmt.Printf("New address is: %s\n", addr)
}

func (cli *CommandLine) listAddrs() {
	ws, _ := wallet.CreateWallets()
	addrs := ws.GetAllAddresses()

	for _, addr := range addrs {
		fmt.Println(addr)
	}
}

func (cli *CommandLine) reindexUTXO() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()

	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	utxoSet.Reindex()

	count := utxoSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createbc", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddrsCmd := flag.NewFlagSet("listaddrs", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The created blockchain")
	sendFrom := sendCmd.String("from", "", "source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount sent")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	case "createbc":
		err := createBlockchainCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	case "listaddrs":
		err := listAddrsCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		handlers.HandleErr(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if listAddrsCmd.Parsed() {
		cli.listAddrs()
	}
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO()
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

}
