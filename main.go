package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

const dateFormat = "20060201    150405" // 18 chars, as per SIP2 spec

var numWorkers = flag.Int("w", 100, "number of concurrent workers")
var workerSleep = flag.Int("s", 1000, "max worker sleep in ms between requests")

var borrowers []string
var items []string
var onLoan map[string]string

func checkout() string {
	b := borrowers[rand.Intn(len(borrowers))]
	i := items[rand.Intn(len(items))]
	onLoan[i] = b // store items on loan to check it in later
	date := time.Now().Format(dateFormat)
	return fmt.Sprintf("11YN%s%sAOHTUL|AA%v|AB%v|ACSTRESS|\r", date, date, b, i)
}

func checkin() string {
	var i string
	for k, _ := range onLoan {
		i = k
		delete(onLoan, k)
		break
	}
	date := time.Now().Format(dateFormat)
	return fmt.Sprintf("09N%s%sAPHUTL|AOHUTL|AB%s|ACSTRESS|\r", date, date, i)
}

func randomRequest() string {
	r := rand.Intn(100)
	if r > 50 {
		return checkin()
	}
	return checkout()
}

func doRequest(w int) {
	conn, err := net.Dial("tcp", "10.172.2.160:6001")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// login
	loginRequest := fmt.Sprintf("9300CNstresstest%d|COstresstest%d|CPHUTL|\r", w, w)
	//fmt.Println("--> " + loginRequest)
	_, err = conn.Write([]byte(loginRequest))
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	_, err = reader.ReadString('\r')
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("<--" + string(loginResponse))

	for {
		sipRequest := randomRequest()
		fmt.Printf("--> %s\n", sipRequest)
		_, err = conn.Write([]byte(sipRequest))
		if err != nil {
			log.Fatal(err)
		}

		res, err := reader.ReadString('\r')
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("<--" + string(res)[1:])
		r := rand.Intn(*workerSleep)
		time.Sleep(time.Duration(r) * time.Millisecond)
	}
}

func init() {
	f1, err := os.Open("borrowers.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f1.Close()
	reader := bufio.NewReader(f1)
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}
	borrowers = strings.Split(string(contents), "\n")

	f2, err := os.Open("items.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f2.Close()
	reader2 := bufio.NewReader(f2)
	if err != nil {
		log.Fatal(err)
	}
	for {
		str, err := reader2.ReadString('\n')
		if err == io.EOF {
			break
		}
		items = append(items, strings.TrimRight(str, "\n"))
	}

	onLoan = make(map[string]string, 1000)
}

func main() {
	flag.Parse()
	for i := 0; i < *numWorkers; i++ {
		go doRequest(i + 1)
	}

	time.Sleep(time.Minute)
}
