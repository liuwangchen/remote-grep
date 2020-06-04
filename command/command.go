package command

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/liuwangchen/remote-grep/console"
	"github.com/liuwangchen/remote-grep/ssh"
)

type Command struct {
	Host   string
	User   string
	Script string
	Stdout io.Reader
	Stderr io.Reader
	Server Server
}

// Message The message used by channel to transport log line by line
type Message struct {
	Host    string
	Content string
}

// NewCommand Create a new command
func NewCommand(server Server) (cmd *Command) {
	return &Command{
		Host:   server.Hostname,
		User:   server.User,
		Script: getGrepScript(server.Searchs, server.TailFile),
		Server: server,
	}
}

func getGrepScript(searchs []string, file string) string {
	script := fmt.Sprintf("grep '%s' --color=auto %s", strings.ReplaceAll(strings.ReplaceAll(searchs[0], "[", "\\["), "]", "\\]"), file)
	if len(searchs) > 1 {
		searchs = searchs[1:]
		for _, search := range searchs {
			script += fmt.Sprintf("| grep '%s' --color=auto", strings.ReplaceAll(strings.ReplaceAll(search, "[", "\\["), "]", "\\]"))
		}
	}
	return script
}

// Execute the remote command
func (cmd *Command) Execute(outputs, errputs chan Message) error {
	defer func() {
		close(outputs)
		close(errputs)
	}()
	client := &ssh.Client{
		Host:           cmd.Host + ":22",
		User:           cmd.User,
		Password:       cmd.Server.Password,
		PrivateKeyPath: cmd.Server.PrivateKeyPath,
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("[%s] unable to connect: %s", cmd.Host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("[%s] unable to create session: %s", cmd.Host, err)
	}
	defer session.Close()

	if err := session.RequestPty("xterm", 80, 40, *ssh.CreateTerminalModes()); err != nil {
		return fmt.Errorf("[%s] unable to create pty: %v", cmd.Host, err)
	}

	cmd.Stdout, err = session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("[%s] redirect stdout failed: %s", cmd.Host, err)
	}

	cmd.Stderr, err = session.StderrPipe()
	if err != nil {
		return fmt.Errorf("[%s] redirect stderr failed: %s", cmd.Host, err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go bindOutput(wg, cmd.Host, outputs, &cmd.Stdout, "", 0)
	go bindOutput(wg, cmd.Host, errputs, &cmd.Stderr, "Error:", console.TextRed)

	if err = session.Start(cmd.Script); err != nil {
		return fmt.Errorf("[%s] failed to execute command: %s", cmd.Host, err)
	}

	if err = session.Wait(); err != nil {
		return fmt.Errorf("[%s] failed to wait command: %s", cmd.Host, err)
	}
	wg.Wait()
	return nil
}

// bing the pipe output for formatted output to channel
func bindOutput(wg *sync.WaitGroup, host string, output chan<- Message, input *io.Reader, prefix string, color int) {
	defer wg.Done()
	reader := bufio.NewReader(*input)
	for {
		line, err := reader.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}

		line = prefix + line
		if color != 0 {
			line = console.ColorfulText(color, line)
		}

		output <- Message{
			Host:    host,
			Content: line,
		}
	}
}
