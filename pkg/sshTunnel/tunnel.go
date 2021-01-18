package sshtunnel

/*
https://gist.github.com/svett/5d695dcc4cc6ad5dd275

*/

import (
	"fmt"
	"github.com/goph/emperror"
	"github.com/op/go-logging"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net"
)

type Endpoint struct {
	Host string
	Port int
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type SSHtunnel struct {
	Local  *Endpoint
	Server *Endpoint
	Remote *Endpoint
	Config *ssh.ClientConfig
	Client *ssh.Client
	log    *logging.Logger
}

func (tunnel *SSHtunnel) String() string {
	return fmt.Sprintf("%v@%v:%v - %v:%v -> %v:%v",
		tunnel.Config.User,
		tunnel.Server.Host, tunnel.Server.Port,
		tunnel.Local.Host, tunnel.Local.Port,
		tunnel.Remote.Host, tunnel.Remote.Port,
	)
}

func (tunnel *SSHtunnel) Start() error {
	tunnel.log.Infof("starting ssh connection listener")

	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go tunnel.forward(conn)
	}
}

func (tunnel *SSHtunnel) forward(localConn net.Conn) {
	var err error

	if tunnel.Client == nil {
		tunnel.log.Infof("dialing ssh: %v", tunnel.String())
		tunnel.Client, err = ssh.Dial("tcp", tunnel.Server.String(), tunnel.Config)
		if err != nil {
			tunnel.log.Errorf("Server dial error: %s\n", err)
			return
		}
	} else {
		tunnel.log.Infof("reusing ssh: %v", tunnel.String())
	}
	remoteConn, err := tunnel.Client.Dial("tcp", tunnel.Remote.String())
	if err != nil {
		fmt.Printf("Remote dial error: %s\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)

}

func NewSSHTunnel(user, privateKey string, localEndpoint, serverEndpoint, remoteEndpoint *Endpoint, log *logging.Logger) (*SSHtunnel, error) {
	key, err := ioutil.ReadFile(privateKey)
	if err != nil {
		return nil, emperror.Wrapf(err, "Unable to read private key %s", privateKey)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, emperror.Wrapf(err, "Unable to parse private key")
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	tunnel := &SSHtunnel{
		Config: sshConfig,
		Local:  localEndpoint,
		Server: serverEndpoint,
		Remote: remoteEndpoint,
		log:    log,
	}

	return tunnel, nil
}
