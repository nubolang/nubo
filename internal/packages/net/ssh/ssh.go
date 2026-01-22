package ssh

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	"golang.org/x/crypto/ssh"
)

var sshStruct *language.Struct

func NewSSH(dg *debug.Debug) language.Object {
	instance := n.NewPackage("ssh", dg)
	proto := instance.GetPrototype()

	if sshStruct == nil {
		sshStruct = language.NewStruct("SSH", []language.StructField{}, dg)

		ctx := context.Background()
		proto := sshStruct.GetPrototype().(*language.StructPrototype)
		proto.Unlock()
		defer proto.Lock()

		proto.SetObject(ctx, "init", n.Function(
			n.Describe(
				n.Arg("self", sshStruct.Type()),
				n.Arg("host", n.TString),
				n.Arg("port", n.TInt),
				n.Arg("user", n.TString),
				n.Arg("shell", n.TBool, n.Bool(true)),
			).Returns(sshStruct.Type()),
			initSSH,
		))
	}

	ctx := context.Background()
	proto.SetObject(ctx, "SSH", sshStruct)

	return instance
}

func initSSH(args *n.Args) (any, error) {
	self := args.Name("self").(*language.StructInstance)
	host := args.Name("host").String()
	port := int(args.Name("port").Value().(int64))
	user := args.Name("user").String()
	useShell := args.Name("shell").Value().(bool)

	ctx := context.Background()
	proto := self.GetPrototype().(*language.StructPrototype)
	proto.Unlock()
	defer proto.Lock()

	var (
		session         *ssh.Session
		client          *ssh.Client
		stdin           io.WriteCloser
		stdout          io.Reader
		state           bool
		errNotConnected = fmt.Errorf("not connected")
	)

	proto.SetObject(ctx, "connect", n.Function(
		n.Describe(
			n.Arg("password", n.Nullable(n.TString), language.Nil), // optional
			n.Arg("keyFile", n.Nullable(n.TString), language.Nil),  // optional PEM/private key
			n.Arg("timeoutMs", n.TInt, language.NewInt(5000, self.Debug())),
		),
		func(a *n.Args) (any, error) {
			password := a.Name("password")
			keyFile := a.Name("keyFile")
			timeout := time.Duration(a.Name("timeoutMs").Value().(int64)) * time.Millisecond

			var authMethods []ssh.AuthMethod
			if keyFile.Type().Compare(n.TString) {
				keyAuth, err := publicKeyFile(keyFile.String())
				if err != nil {
					return nil, fmt.Errorf("invalid key file: %w", err)
				}
				authMethods = append(authMethods, keyAuth)
			}
			if password.Type().Compare(n.TString) {
				authMethods = append(authMethods, ssh.Password(password.String()))
			}

			config := &ssh.ClientConfig{
				User:            user,
				Auth:            authMethods,
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         timeout,
			}

			addr := fmt.Sprintf("%s:%d", host, port)

			var err error
			client, err = ssh.Dial("tcp", addr, config)
			if err != nil {
				return nil, fmt.Errorf("ssh dial failed: %w", err)
			}

			session, err = client.NewSession()
			if err != nil {
				client.Close()
				return nil, fmt.Errorf("cannot create session: %w", err)
			}

			stdin, err = session.StdinPipe()
			if err != nil {
				session.Close()
				client.Close()
				return nil, fmt.Errorf("cannot get stdin: %w", err)
			}

			stdout, err = session.StdoutPipe()
			if err != nil {
				session.Close()
				client.Close()
				return nil, fmt.Errorf("cannot get stdout: %w", err)
			}

			if useShell {
				if err := session.Shell(); err != nil {
					session.Close()
					client.Close()
					return nil, fmt.Errorf("cannot start shell: %w", err)
				}
			}

			state = true

			return nil, nil
		},
	))

	// Provide interactive methods
	proto.SetObject(ctx, "write", n.Function(
		n.Describe(n.Arg("data", n.TString)),
		func(a *n.Args) (any, error) {
			if !state {
				return nil, errNotConnected
			}

			data := a.Name("data").String()
			_, err := stdin.Write([]byte(data))
			return nil, err
		},
	))

	proto.SetObject(ctx, "read", n.Function(
		n.Describe(n.Arg("timeoutMs", n.TInt)).Returns(n.TString),
		func(a *n.Args) (any, error) {
			if !state {
				return nil, errNotConnected
			}

			reader := bufio.NewReader(stdout)
			timeout := time.Duration(a.Name("timeoutMs").Value().(int64)) * time.Millisecond

			lineCh := make(chan string)
			errCh := make(chan error)
			go func() {
				line, err := reader.ReadString('\n')
				if err != nil {
					errCh <- err
				} else {
					lineCh <- line
				}
			}()

			select {
			case line := <-lineCh:
				return n.String(line, a.Name("timeoutMs").Debug()), nil
			case err := <-errCh:
				return nil, err
			case <-time.After(timeout):
				return n.String("", a.Name("timeoutMs").Debug()), nil
			}
		},
	))

	proto.SetObject(ctx, "run", n.Function(
		n.Describe(n.Arg("cmd", n.TString)).Returns(n.TString),
		func(a *n.Args) (any, error) {
			if !state {
				return nil, errNotConnected
			}

			cmd := a.Name("cmd").String()
			out, err := session.CombinedOutput(cmd)
			return n.String(string(out), a.Name("cmd").Debug()), err
		},
	))

	proto.SetObject(ctx, "state", n.Function(
		n.Describe().Returns(n.TBool),
		func(a *n.Args) (any, error) {
			return n.Bool(state), nil
		},
	))

	proto.SetObject(ctx, "close", n.Function(
		n.Describe(),
		func(a *n.Args) (any, error) {
			if !state {
				return nil, errNotConnected
			}

			defer func() { state = false }()

			session.Close()
			return nil, client.Close()
		},
	))

	return self, nil
}
