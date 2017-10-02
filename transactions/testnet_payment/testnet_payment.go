package main

import (
    "fmt"
    "log"
    "flag"
    "strings"
    "net/http"
    "github.com/stellar/go/keypair"
    "github.com/stellar/go/clients/horizon"
    builder "github.com/stellar/go/build"
)

const baseUrlDefault = "https://horizon-testnet.stellar.org"
const baseUrlLocal = "http://localhost:8000"

func main() {
    localPtr := flag.Bool("l", false, "(optional) whether we should use the local horizon server @ " + baseUrlLocal)
    fromSeedPtr := flag.String("fromSeed", "", "seed of the sender's account")
    toAddressPtr := flag.String("toAddress", "", "destination address of the receiver's account")
    amountPtr := flag.Float64("amount", 0.0, "amount to be sent, must be > 0.0")
    memoPtr := flag.String("memo", "", "(optional) memo to include with the payment")
    assetPtr := flag.String("asset", "", "(optional) asset to pay with, of the form code:issuer")
    flag.Parse()

    if *fromSeedPtr == "" || *toAddressPtr == "" || *amountPtr <= 0 {
        flag.PrintDefaults()
        return
    }

    baseUrl := baseUrlDefault
    if *localPtr {
        baseUrl = baseUrlLocal
    }

    sourceSeed := *fromSeedPtr
    destinationAddress := *toAddressPtr
    amount := *amountPtr
    memo := *memoPtr
    asset := *assetPtr
    sourceKP, err := keypair.Parse(sourceSeed)
    if err != nil {
        log.Fatal(err)
    }
    sourceAddress := sourceKP.Address()

    fmt.Println("local:", *localPtr)
    fmt.Println("baseUrl:", baseUrl)
    fmt.Println("fromSeed:", sourceSeed)
    fmt.Println("fromAddress:", sourceAddress)
    fmt.Println("toAddress:", destinationAddress)
    fmt.Println("amount:", amount)
    fmt.Println("memo:", memo)
    fmt.Println("asset:", asset)
    fmt.Println()

    horizonClient := &horizon.Client{
        URL: baseUrl,
        HTTP: http.DefaultClient,
    }

    // validate accounts
    loadAccount(horizonClient, sourceAddress, "source")
    loadAccount(horizonClient, destinationAddress, "destination")

    amountStr := fmt.Sprintf("%v", amount)
    var assetAmount builder.PaymentMutator
    if asset != "" {
        assetParts := strings.SplitN(asset, ":", 2)
        assetAmount = builder.CreditAmount{assetParts[0], assetParts[1], amountStr}
        fmt.Println("using non-native asset:", assetAmount)
    } else {
        assetAmount = builder.NativeAmount{amountStr}
    }

    txn := builder.Transaction(
        builder.SourceAccount{sourceSeed},
        builder.AutoSequence{horizonClient},
        builder.TestNetwork,
        builder.Payment(
            builder.Destination{destinationAddress},
            assetAmount,
        ),
    )
    if memo != "" {
        txn.Mutate(builder.MemoText{memo})
    }
    // sign
    txnS := txn.Sign(sourceSeed)
    // convert to base64
    txnS64, err2 := txnS.Base64()
    if err2 != nil {
        log.Fatal(err2)
    }
    fmt.Printf("tx base64: %s\n", txnS64)

    // submit the transaction
    resp, err3 := horizonClient.SubmitTransaction(txnS64)
    if err3 != nil {
        log.Fatal(err3)
    }
    fmt.Println("transaction posted in ledger:", resp.Ledger)

    // print final balances by reloading accounts
    loadAccount(horizonClient, sourceAddress, "source")
    loadAccount(horizonClient, destinationAddress, "destination")
}

func loadAccount(horizonClient *horizon.Client, publicKey string, accountName string) horizon.Account {
    account, err := horizonClient.LoadAccount(publicKey)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Native Balance for account (" + accountName + "):", account.GetNativeBalance())
    return account
}
