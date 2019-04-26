package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/jamescun/tuntap"
)

// 16384

// MTU é o MTU das interfaces TAP
var MTU int

func sigHandler(ctx context.Context, cancel context.CancelFunc, chanSig chan os.Signal, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return
		case sig := <-chanSig:
			switch sig {
			case syscall.SIGHUP:
				cancel()
				wg.Done()
				return
			case syscall.SIGINT:
				cancel()
				wg.Done()
				return
			case syscall.SIGQUIT:
				cancel()
				wg.Done()
				return
			}
		}
	}
}

func forwarder(src io.Reader, dst io.Writer, bufsize uint16) {
	data := make([]byte, bufsize)
	for {
		_, err := src.Read(data)
		if err != nil {
			return
		}
		_, err = dst.Write(data)
		if err != nil {
			return
		}
	}
}

func getTap(ctx context.Context, ip, name string, tuntype string) (tuntap.Interface, error) {
	var tap tuntap.Interface
	var err error

	if tuntype == "tun" {
		tap, err = tuntap.Tun(name)
		if err != nil {
			return nil, err
		}
	} else if tuntype == "tap" {
		tap, err = tuntap.Tap(name)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("Configurando o IP %s da interface %s\n", ip, tap.Name())
	cmd := exec.CommandContext(ctx, "ip", "addr", "add", ip, "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	log.Printf("Subindo a interface %s\n", tap.Name())
	cmd = exec.CommandContext(ctx, "ip", "link", "set", "up", "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	log.Printf("Configurando a interface %s com MTU de %d bytes\n", tap.Name(), MTU)
	cmd = exec.CommandContext(ctx, "ip", "link", "set", "mtu", strconv.Itoa(MTU), "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	return tap, nil
}

func configClient(ctx context.Context, cancel context.CancelFunc, pair string, tuntype string) {
	tap, err := getTap(ctx, "10.253.0.2/30", "", tuntype)
	if err != nil {
		log.Println(err)
		cancel()
		return
	}

	log.Printf("Efetuando conexão para %s\n", pair)
	conn, err := net.Dial("tcp", pair)
	if err != nil {
		log.Println(err)
		cancel()
		return
	}

	log.Printf("Conexão efetuada com sucesso\n")

	go forwarder(tap, conn, uint16(MTU))
	go forwarder(conn, tap, uint16(MTU))
}

func configServer(ctx context.Context, cancel context.CancelFunc, pair string, tuntype string) {
	tap, err := getTap(ctx, "10.253.0.1/30", "", tuntype)
	if err != nil {
		log.Println(err)
		cancel()
		return
	}

	ln, err := net.Listen("tcp", pair)
	if err != nil {
		log.Println("Erro ao tentar criar socket listen", err)
		cancel()
		return
	}

	log.Printf("Esperando por conexão...")
	conn, err := ln.Accept()
	if err != nil {
		log.Println("Erro ao tentar aceitar conexão", err)
		cancel()
		return
	}

	log.Printf("Conexão aceita: %s\n", conn.RemoteAddr().String())

	go forwarder(tap, conn, uint16(MTU))
	go forwarder(conn, tap, uint16(MTU))
}

func main() {
	var wg sync.WaitGroup

	s := flag.Bool("s", false, "Use essa flag para escutar por uma conexão")
	c := flag.Bool("c", false, "Use essa flag para efetuar uma conexão")
	pair := flag.String("ip", "0.0.0.0:80", "Ip e porta para se conectar ou escutar, no formato <ip>:<porta>")
	mtu := flag.Int("mtu", 1500, "Define o mtu da interface tun")
	tuntype := flag.String("tun-type", "tun", "Define se a interface criada será tun ou tap")

	flag.Parse()

	if strings.ToUpper(*tuntype) != "TUN" && strings.ToUpper(*tuntype) != "TAP" {
		panic("O tipo de interface deve ser tun ou tap!")
	}

	if *mtu < 0 {
		panic("O MTU da interface não pode ser negativo!")
	}

	MTU = *mtu

	if *s && *c {
		panic("O programa só pode ser comportar ou como servidor ou cliente por vez.\nExecute passando -help como parâmetro para mais informações.")
	}
	if !*s && !*c {
		panic("Defina se o programa se comportará como servidor ou cliente.\nExecute passando -help como parâmetro para mais informações.")
	}

	ctx, cancel := context.WithCancel(context.Background())

	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

	wg.Add(1)

	go sigHandler(ctx, cancel, chanSig, &wg)

	if *s {
		go configServer(ctx, cancel, *pair, strings.ToLower(*tuntype))
	} else {
		go configClient(ctx, cancel, *pair, strings.ToLower(*tuntype))
	}

	log.Println("Esperando finalização...")
	wg.Wait()
}
