// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"encoding/json"
	"fmt"
	"time"

	"frp/src/frp/models/consts"
	"frp/src/frp/models/msg"
	"frp/src/frp/utils/conn"
	"frp/src/frp/utils/log"
	"frp/src/frp/utils/pcrypto"
)

type ProxyClient struct {
	Name          string
	AuthToken     string
	LocalIp       string
	LocalPort     int64
	Type          string
	UseEncryption bool
}

func (p *ProxyClient) GetLocalConn() (c *conn.Conn, err error) {
	c, err = conn.ConnectServer(p.LocalIp, p.LocalPort)
	if err != nil {
		log.Error("ProxyName [%s], connect to local port error, %v", p.Name, err)
	}
	return
}

func (p *ProxyClient) GetRemoteConn(addr string, port int64) (c *conn.Conn, err error) {
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	c, err = conn.ConnectServer(addr, port)
	if err != nil {
		log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", p.Name, addr, port, err)
		return
	}

	nowTime := time.Now().Unix()
	authKey := pcrypto.GetAuthKey(p.Name + p.AuthToken + fmt.Sprintf("%d", nowTime))
	req := &msg.ControlReq{
		Type:      consts.NewWorkConn,
		ProxyName: p.Name,
		AuthKey:   authKey,
		Timestamp: nowTime,
	}

	buf, _ := json.Marshal(req)
	err = c.Write(string(buf) + "\n")
	if err != nil {
		log.Error("ProxyName [%s], write to server error, %v", p.Name, err)
		return
	}

	err = nil
	return
}

func (p *ProxyClient) StartTunnel(serverAddr string, serverPort int64) (err error) {
	localConn, err := p.GetLocalConn()
	if err != nil {
		return
	}
	remoteConn, err := p.GetRemoteConn(serverAddr, serverPort)
	if err != nil {
		return
	}

	// l means local, r means remote
	log.Debug("Join two connections, (l[%s] r[%s]) (l[%s] r[%s])", localConn.GetLocalAddr(), localConn.GetRemoteAddr(),
		remoteConn.GetLocalAddr(), remoteConn.GetRemoteAddr())
	if p.UseEncryption {
		go conn.JoinMore(localConn, remoteConn, p.AuthToken)
	} else {
		go conn.Join(localConn, remoteConn)
	}

	return nil
}
