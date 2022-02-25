package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

type CollectInfos struct {
	ClientHellos []*tls.ClientHelloInfo
	sync.Mutex
}

var collectInfos CollectInfos
var currentClientHello *tls.ClientHelloInfo

func (c *CollectInfos) collectClientHello(clientHello *tls.ClientHelloInfo) {
	c.Lock()
	defer c.Unlock()
	c.ClientHellos = append(c.ClientHellos, clientHello)
}

func (c *CollectInfos) DumpInfo() {
	c.Lock()
	defer c.Unlock()
	data, err := json.Marshal(c.ClientHellos)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	err = ioutil.WriteFile("hello.json", data, 0655)
	fmt.Println("writeFile.err", err)
}

func getCert() *tls.Certificate {
	cert, err := tls.LoadX509KeyPair("../cert.pem", "../key.pem")
	if err != nil {
		log.Println(err)
		return nil
	}
	return &cert
}

func buildTlsConfig(cert *tls.Certificate) *tls.Config {
	cfg := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		GetConfigForClient: func(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
			collectInfos.collectClientHello(clientHello)
			currentClientHello = clientHello
			return nil, nil
		},
	}
	return cfg
}

func serve(cfg *tls.Config) {
	ln, err := tls.Listen("tcp", ":4433", cfg)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handler(conn)
	}
}

func handler(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(msg)
		data, err := json.Marshal(currentClientHello)
		if err != nil {
			log.Fatal(err)
		}
		_, err = conn.Write(data)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func main() {
	go func() {
		for {
			time.Sleep(3 * time.Second)
			fmt.Println("dumpInfo")
			collectInfos.DumpInfo()
		}
	}()
	cert := getCert()

	fmt.Println(cert)
	if cert != nil {
		serve(buildTlsConfig(cert))
	}
}
