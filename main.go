package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"

	"github.com/liuwangchen/remote-grep/command"
	"github.com/liuwangchen/remote-grep/console"
)

var mossSep = ".--. --- .-- . .-. . -..   -... -.--   -- -.-- .-.. -..- ... .-- \n"

var welcomeMessage = getWelcomeMessage() + console.ColorfulText(console.TextMagenta, mossSep)

var configFile string
var env string
var label string
var file string
var searchs []string

var Version = "3.0"

func usageAndExit(message string) {

	if message != "" {
		_, _ = fmt.Fprintln(os.Stderr, message)
	}

	flag.Usage()
	fmt.Println("remote-grep 'search' env.label.file")
	_, _ = fmt.Fprint(os.Stderr, "\n")

	os.Exit(1)
}

func printWelcomeMessage() {
	fmt.Println(welcomeMessage)

	for _, server := range viper.GetStringSlice(label) {
		serverInfo := fmt.Sprintf("%s@%s:%s", viper.GetString("user"), server, viper.GetString("file."+file))
		fmt.Println(console.ColorfulText(console.TextMagenta, serverInfo))
	}
	fmt.Printf("\n%s\n", console.ColorfulText(console.TextCyan, mossSep))
}

func main() {

	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, welcomeMessage)
		_, _ = fmt.Fprint(os.Stderr, "Options:\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		usageAndExit("")
	}
	searchs = args[:len(args)-1]
	subArgs := args[len(args)-1]
	env = strings.Split(subArgs, ".")[0]
	label = strings.Split(subArgs, ".")[1]
	file = strings.Split(subArgs, ".")[2]
	homeDir, _ := os.UserHomeDir()
	configFile = filepath.Join(homeDir, ".remote", fmt.Sprintf("%s.yaml", env))
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	printWelcomeMessage()

	outputChanList := make([]<-chan command.Message, 0)
	for _, server := range viper.GetStringSlice(label) {
		outputs := make(chan command.Message, 255)
		errputs := make(chan command.Message, 255)
		outputChanList = append(outputChanList, outputs, errputs)
		go func(server command.Server) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf(console.ColorfulText(console.TextRed, "Error: %s\n"), err)
				}
			}()
			cmd := command.NewCommand(server)
			cmd.Execute(outputs, errputs)
		}(command.Server{
			ServerName:     "",
			Hostname:       server,
			Port:           22,
			User:           viper.GetString("user"),
			Password:       viper.GetString("password"),
			PrivateKeyPath: viper.GetString("private_key_path"),
			TailFile:       viper.GetString("file." + file),
			Searchs:        searchs,
		})
	}
	if len(viper.GetStringSlice(label)) > 0 {
		for output := range mergeChan(outputChanList...) {
			content := strings.Trim(output.Content, "\r\n")
			// 去掉文件名称输出
			if content == "" || (strings.HasPrefix(content, "==>") && strings.HasSuffix(content, "<==")) {
				continue
			}

			fmt.Printf(
				"%s %s %s\n",
				console.ColorfulText(console.TextGreen, output.Host),
				console.ColorfulText(console.TextYellow, "->"),
				content,
			)
		}
	} else {
		fmt.Println(console.ColorfulText(console.TextRed, "No target host is available"))
	}
}

func mergeChan(cs ...<-chan command.Message) <-chan command.Message {
	out := make(chan command.Message)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan command.Message) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func getWelcomeMessage() string {
	return `
 ____                      _
|  _ \ ___ _ __ ___   ___ | |_ ___  __ _ _ __ ___ _ __
| |_) / _ \ /_ \ _ \ / _ \| __/ _ \/ _\ | /__/ _ \ \_ \
|  _ <  __/ | | | | | (_) | ||  __/ (_| | | |  __/ |_) |
|_| \_\___|_| |_| |_|\___/ \__\___|\__, |_|  \___| .__/
                                   |___/         |_|

Author: liuwangchen
Version: ` + Version + `
`
}
