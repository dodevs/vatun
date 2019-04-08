package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
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

func writerFunc(tap tuntap.Interface, conn net.Conn) {
	packet := make([]byte, MTU)
	for {
		_, err := conn.Read(packet)
		if err != nil {
			panic(err)
		}
		_, err = tap.Write(packet)
		if err != nil {
			panic(err)
		}
	}
}

func readerFunc(tap tuntap.Interface, conn net.Conn) {
	packet := make([]byte, MTU)
	for {
		_, err := tap.Read(packet)
		if err != nil {
			panic(err)
		}
		_, err = conn.Write(packet)
		if err != nil {
			panic(err)
		}
	}
}

func configClient(ctx context.Context, cancel context.CancelFunc, tap tuntap.Interface, pair string) {
	log.Printf("Configurando o IP %s da interface %s\n", "10.0.0.2/30", tap.Name())
	cmd := exec.CommandContext(ctx, "ip", "addr", "add", "10.0.0.2/30", "dev", tap.Name())
	err := cmd.Run()
	if err != nil {
		log.Println("Erro ao tentar configurar IP da interface", err)
		cancel()
		return
	}

	log.Printf("Subindo a interface %s\n", tap.Name())
	cmd = exec.CommandContext(ctx, "ip", "link", "set", "up", "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		log.Println("Erro ao tentar subir interface", err)
		cancel()
		return
	}

	log.Printf("Configurando a interface %s com MTU de %d bytes\n", tap.Name(), MTU)
	cmd = exec.CommandContext(ctx, "ip", "link", "set", "mtu", strconv.Itoa(MTU), "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		log.Println("Erro ao tentar configurar MTU da interface", err)
		cancel()
		return
	}

	log.Printf("Efetuando conexão para %s\n", pair)
	conn, err := net.Dial("tcp", pair)
	if err != nil {
		log.Println(err)
		cancel()
	}
	log.Printf("Conexão efetuada com sucesso\n")

	go writerFunc(tap, conn)
	go readerFunc(tap, conn)
}

func configServer(ctx context.Context, cancel context.CancelFunc, tap tuntap.Interface, pair string) {
	log.Printf("Configurando o IP %s da interface %s\n", "10.0.0.1/30", tap.Name())
	cmd := exec.CommandContext(ctx, "ip", "addr", "add", "10.0.0.1/30", "dev", tap.Name())
	err := cmd.Run()
	if err != nil {
		log.Println("Erro ao tentar configurar IP da interface", err)
		cancel()
		return
	}

	log.Printf("Subindo a interface %s\n", tap.Name())
	cmd = exec.CommandContext(ctx, "ip", "link", "set", "up", "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		log.Println("Erro ao tentar configurar MTU da interface", err)
		cancel()
		return
	}

	log.Printf("Configurando a interface %s com MTU de %d bytes\n", tap.Name(), MTU)
	cmd = exec.CommandContext(ctx, "ip", "link", "set", "mtu", strconv.Itoa(MTU), "dev", tap.Name())
	err = cmd.Run()
	if err != nil {
		log.Println("Erro ao tentar configurar MTU da interface", err)
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

	go writerFunc(tap, conn)
	go readerFunc(tap, conn)
}

func main() {
	var wg sync.WaitGroup

	s := flag.Bool("s", false, "Use essa flag para escutar por uma conexão")
	c := flag.Bool("c", false, "Use essa flag para efetuar uma conexão")
	pair := flag.String("ip", "0.0.0.0:80", "Ip e porta para se conectar ou escutar, no formato <ip>:<porta>")
	mtu := flag.Int("mtu", 1500, "Define o mtu da interface tun")

	flag.Parse()

	if *mtu < 0 {
		panic("O MTU da interface não pode ser negativo!")
	}

	MTU = *mtu

	if (*s && *c) || (!*s && !*c) {
		panic("O programa só pode ser comportar ou como servidor ou cliente por vez")
	}

	ctx, cancel := context.WithCancel(context.Background())

	chanSig := make(chan os.Signal, 1)
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

	wg.Add(1)

	go sigHandler(ctx, cancel, chanSig, &wg)

	tap, err := tuntap.Tun("")
	if err != nil {
		log.Println(err)
		cancel()
		return
	}

	if *s {
		go configServer(ctx, cancel, tap, *pair)
	} else {
		go configClient(ctx, cancel, tap, *pair)
	}

	log.Println("Esperando finalização...")
	wg.Wait()
}
