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
	"github.com/FG420/go-block/network"
	"github.com/FG420/go-block/wallet"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage: ")
	fmt.Println(" getbalance -addr ADDRESS - get the balance of the address")
	fmt.Println(" createbc -addr ADDRESS - Creates a blockchain")
	fmt.Println(" printchain - Prints the blocks in the chain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT -mine - Send amount of coins. Then -mine flag enables the mining of that transaction")
	fmt.Println(" createwallet - Creates a new Wallet")
	fmt.Println(" listaddrs - List the addresses in our wallet file ")
	fmt.Println(" reindexutxo - Rebuild the UTXO set ")
	fmt.Println(" startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env -miner enables mining ")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
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

func (cli *CommandLine) createBlockChain(addr, nodeId string) {
	if !wallet.ValidateAddress(addr) {
		log.Panic("Address in not valid")
	}

	chain := blockchain.InitBlockChain(addr, nodeId)
	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	utxoSet.Reindex()
	chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(addr, nodeId string) {
	if !wallet.ValidateAddress(addr) {
		log.Panic("Address in not valid")
	}

	chain := blockchain.ContinueBlockChain(nodeId)
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

func (cli *CommandLine) send(from, to string, amount int, nodeId string, mineNow bool) {
	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		log.Panic("Address in not valid")
	}

	chain := blockchain.ContinueBlockChain(nodeId)
	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeId)
	handlers.HandleErr(err)
	wallet := wallets.GetAddress(from)

	log.Print("initialize new Transaction")
	tx := blockchain.NewTransaction(wallet, to, amount, &utxoSet)
	if mineNow {
		log.Println("mine: true")
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		utxoSet.Update(block)
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("tx sent")
	}

	fmt.Println("Success!")
}

// func (cli *CommandLine) send(from, to string, amount int) {
// 	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
// 		log.Panic("Address in not valid")
// 	}

// 	chain := blockchain.ContinueBlockChain(from)
// 	utxoSet := blockchain.UTXOSet{BlockChain: chain}
// 	defer chain.Database.Close()

// 	log.Print("initialize new Transaction")
// 	tx := blockchain.NewTransaction(from, to, amount, &utxoSet)
// 	cbTx := blockchain.CoinbaseTx(from, "")
// 	block := chain.AddBlock([]*blockchain.Transaction{cbTx, tx})

// 	utxoSet.Update(block)
// 	fmt.Println("Success!")
// }

func (cli *CommandLine) createWallet(nodeId string) {
	ws, _ := wallet.CreateWallets(nodeId)
	addr := ws.AddWallet()

	ws.SaveFile(nodeId)

	fmt.Printf("New address is: %s\n", addr)
}

func (cli *CommandLine) listAddrs(nodeId string) {
	ws, _ := wallet.CreateWallets(nodeId)
	addrs := ws.GetAllAddresses()

	for _, addr := range addrs {
		fmt.Println(addr)
	}
}

func (cli *CommandLine) reindexUTXO(nodeId string) {
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()

	utxoSet := blockchain.UTXOSet{BlockChain: chain}
	utxoSet.Reindex()

	count := utxoSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) StartNode(nodeId, minerAddr string) {
	fmt.Printf("Starting node %s\n", nodeId)

	if len(minerAddr) > 0 {
		if wallet.ValidateAddress(minerAddr) {
			fmt.Println("Mining is on. Addr to receive rewards: ", minerAddr)
		} else {
			log.Panic("Wrong miner address.")
		}
	}
	network.StartServer(nodeId, minerAddr)
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createbc", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddrsCmd := flag.NewFlagSet("listaddrs", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("addr", "", "The address")
	createBlockchainAddress := createBlockchainCmd.String("addr", "", "The created blockchain")
	sendFrom := sendCmd.String("from", "", "source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount sent")
	sendMine := sendCmd.Bool("mine", false, "Mine immidiately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining node and send reward to the miner")

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
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
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
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress, nodeID)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddrsCmd.Parsed() {
		cli.listAddrs(nodeID)
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

}
