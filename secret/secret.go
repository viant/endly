package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
)

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.String("k", path.Join(os.Getenv("HOME"), ".secret/id_rsa"), "<path to private key>")
	flag.Bool("h", false, "print help")
	flag.Bool("v", true, "print version")
	flag.String("o", "", "secret name")
}

func printVersion() {
	fmt.Fprintf(os.Stdout, "%v.secret %v\n", endly.AppName, endly.GetVersion())
}

func printHelp() {
	_, name := path.Split(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", name)
	fmt.Fprintf(os.Stderr, "secret [options] \n")
	fmt.Fprintf(os.Stderr, "where options include:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printHelp
	flag.Parse()
	flagset := make(map[string]string)
	flag.Visit(func(f *flag.Flag) {
		flagset[f.Name] = f.Value.String()
	})

	_, shouldQuit := flagset["v"]
	flagset["v"] = flag.Lookup("v").Value.String()

	if toolbox.AsBoolean(flagset["v"]) {
		printVersion()
		if shouldQuit {
			return
		}
	}

	if _, ok := flagset["h"]; ok {
		printHelp()
		return
	}

	outputFile, ok := flagset["o"]
	if !ok {
		fmt.Printf("-o was migging\n")
		printHelp()
		return
	}

	var secretPath = path.Join(os.Getenv("HOME"), ".secret")
	if !toolbox.FileExists(secretPath) {
		os.Mkdir(secretPath, 0744)
	}
	username, password, err := credentials()
	if err != nil {
		fmt.Printf("\n")
		log.Fatal(err)
	}
	fmt.Println("")
	config := &cred.Config{
		Username: username,
		Password: password,
	}
	var privateKeyPath = flag.Lookup("k").Value.String()
	if toolbox.FileExists(privateKeyPath) && !cred.IsKeyEncrypted(privateKeyPath) {
		config.PrivateKeyPath = privateKeyPath
	}
	var secretFile = path.Join(secretPath, fmt.Sprintf("%v.json", outputFile))
	err = config.Save(secretFile)
	if err != nil {
		fmt.Printf("\n")
		log.Fatal(err)
	}
}

func credentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')
	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("failed to read password %v", err)
	}
	fmt.Print("\nRetype Password: ")
	bytePassword2, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("failed to read password %v", err)
	}

	password := string(bytePassword)
	if string(bytePassword2) != password {
		return "", "", errors.New("Password did not match")
	}
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}
